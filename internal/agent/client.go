package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"reflect"

	m "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/retry"
	"github.com/go-resty/resty/v2"
)

type Client struct {
	client *resty.Client
	logger *slog.Logger
}

func NewClient(baseURL string, logger *slog.Logger) *Client {
	client := resty.New().
		SetBaseURL(baseURL).
		SetRetryCount(0)

	return &Client{
		client: client,
		logger: logger,
	}
}

func isRetriableNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

func (c *Client) post(ctx context.Context, path string, headers map[string]string, body any) error {
	return retry.Do(ctx, isRetriableNetworkError, func() error {
		req := c.client.R()
		for k, v := range headers {
			req.SetHeader(k, v)
		}
		resp, err := req.SetBody(body).Post(path)
		if err != nil {
			return err
		}
		if !resp.IsSuccess() {
			return fmt.Errorf("unexpected status code %d", resp.StatusCode())
		}
		return nil
	})
}

func (c *Client) SendGauge(name string, value float64) error {
	body := m.NewGauge(name, value)
	if err := c.post(context.Background(), "/update", map[string]string{"Content-Type": "application/json"}, body); err != nil {
		return fmt.Errorf("failed to send gauge %s: %w", name, err)
	}
	c.logger.Debug("gauge sent", slog.String("name", name), slog.Float64("value", value))
	return nil
}

func (c *Client) SendCounter(name string, value int64) error {
	body := m.NewCounter(name, value)
	if err := c.post(context.Background(), "/update", map[string]string{"Content-Type": "application/json"}, body); err != nil {
		return fmt.Errorf("failed to send counter %s: %w", name, err)
	}
	c.logger.Debug("counter sent", slog.String("name", name), slog.Int64("value", value))
	return nil
}

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
	if err := c.post(ctx, "/updates/", headers, buf.Bytes()); err != nil {
		return fmt.Errorf("failed to send metrics batch: %w", err)
	}

	c.logger.Debug("metrics batch sent", slog.Int("count", len(metrics)))
	return nil
}

func (c *Client) SendGaugeMetrics(metrics GaugeMetrics) error {
	v := reflect.ValueOf(metrics)
	t := reflect.TypeOf(metrics)

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := field.Type()
		fieldName := t.Field(i).Name

		if !field.CanInterface() {
			continue
		}

		var err error
		switch fieldType.Kind() {
		case reflect.Float64:
			err = c.SendGauge(fieldName, field.Float())
		default:
			c.logger.Warn("unsupported field type",
				slog.String("field", fieldName),
				slog.String("type", fieldType.String()))
			continue
		}

		if err != nil {
			c.logger.Error("failed to send gauge metric",
				slog.String("field", fieldName),
				slog.String("error", err.Error()))
		}
	}

	return nil
}

func (c *Client) SendCounterMetrics(metrics CountMetrics) error {
	err := c.SendCounter("PollCount", metrics.PollCount)

	if err != nil {
		c.logger.Error("failed to send counter metric", "PollCount", slog.String("error", err.Error()))
	}
	return nil
}
