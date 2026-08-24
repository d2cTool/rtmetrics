package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/d2cTool/rtmetrics/internal/hash"
	m "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/retry"
	"github.com/d2cTool/rtmetrics/internal/rsaenc"
	"github.com/go-resty/resty/v2"
)

// Client — HTTP-клиент агента: JSON, gzip для батча, опциональная подпись HashSHA256
// и опциональное RSA-шифрование тела.
type Client struct {
	client *resty.Client
	logger *slog.Logger
	key    string
	pub    *rsa.PublicKey
}

// NewClient создаёт клиента агента. Непустой key включает подпись запросов
// заголовком HashSHA256.
func NewClient(baseURL, key string, logger *slog.Logger) *Client {
	client := resty.New().
		SetBaseURL(baseURL).
		SetRetryCount(0)

	return &Client{
		client: client,
		logger: logger,
		key:    key,
	}
}

// WithPublicKey включает шифрование тел запросов публичным ключом сервера.
func (c *Client) WithPublicKey(pub *rsa.PublicKey) *Client {
	c.pub = pub
	return c
}

func isRetriableNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

// post отправляет body, подписывая signPayload. Для сжатых запросов это разные
// байты: сервер считает хеш после распаковки, поэтому подписывать нужно
// исходный JSON, а не gzip-поток.
func (c *Client) post(ctx context.Context, path string, headers map[string]string, body, signPayload []byte) error {
	return retry.Do(ctx, isRetriableNetworkError, func() error {
		req := c.client.R().SetContext(ctx)
		for k, v := range headers {
			req.SetHeader(k, v)
		}
		if c.key != "" {
			req.SetHeader(hash.Header, hash.Sign(signPayload, c.key))
		}
		wire := body
		if c.pub != nil {
			enc, err := rsaenc.Encrypt(c.pub, body)
			if err != nil {
				return err
			}
			wire = enc
			req.SetHeader(rsaenc.Header, rsaenc.HeaderValue)
		}
		resp, err := req.SetBody(wire).Post(path)
		if err != nil {
			return err
		}
		if !resp.IsSuccess() {
			return fmt.Errorf("unexpected status code %d", resp.StatusCode())
		}
		return nil
	})
}

// SendGauge отправляет одну gauge на POST /update.
func (c *Client) SendGauge(ctx context.Context, name string, value float64) error {
	body, err := json.Marshal(m.NewGauge(name, value))
	if err != nil {
		return fmt.Errorf("failed to marshal gauge %s: %w", name, err)
	}
	if err := c.post(ctx, "/update", map[string]string{"Content-Type": "application/json"}, body, body); err != nil {
		return fmt.Errorf("failed to send gauge %s: %w", name, err)
	}
	c.logger.Debug("gauge sent", slog.String("name", name), slog.Float64("value", value))
	return nil
}

// SendCounter отправляет один counter на POST /update.
func (c *Client) SendCounter(ctx context.Context, name string, value int64) error {
	body, err := json.Marshal(m.NewCounter(name, value))
	if err != nil {
		return fmt.Errorf("failed to marshal counter %s: %w", name, err)
	}
	if err := c.post(ctx, "/update", map[string]string{"Content-Type": "application/json"}, body, body); err != nil {
		return fmt.Errorf("failed to send counter %s: %w", name, err)
	}
	c.logger.Debug("counter sent", slog.String("name", name), slog.Int64("value", value))
	return nil
}

// SendBatch сжимает метрики gzip и шлёт их на POST /updates/.
func (c *Client) SendBatch(ctx context.Context, metrics []m.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics batch: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return fmt.Errorf("failed to gzip metrics batch: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	headers := map[string]string{
		"Content-Type":     "application/json",
		"Content-Encoding": "gzip",
	}
	if err := c.post(ctx, "/updates/", headers, buf.Bytes(), data); err != nil {
		return fmt.Errorf("failed to send metrics batch: %w", err)
	}

	c.logger.Debug("metrics batch sent", slog.Int("count", len(metrics)))
	return nil
}

// SendGaugeMetrics отправляет каждую gauge отдельным POST /update.
func (c *Client) SendGaugeMetrics(ctx context.Context, metrics GaugeMetrics) error {
	for _, metric := range metrics.Gauges() {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := c.SendGauge(ctx, metric.ID, *metric.Value); err != nil {
			c.logger.Error("failed to send gauge metric",
				slog.String("field", metric.ID),
				slog.String("error", err.Error()))
		}
	}

	return nil
}

// SendCounterMetrics отправляет PollCount на POST /update.
func (c *Client) SendCounterMetrics(ctx context.Context, metrics CountMetrics) error {
	err := c.SendCounter(ctx, "PollCount", metrics.PollCount)

	if err != nil {
		c.logger.Error("failed to send counter metric", "PollCount", slog.String("error", err.Error()))
	}
	return nil
}
