package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"

	m "github.com/d2cTool/rtmetrics/internal/model"
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

func (c *Client) SendGauge(name string, value float64) error {
	body := m.NewGauge(name, value)
	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post("/update")

	if err != nil {
		return fmt.Errorf("failed to send gauge %s: %w", name, err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("unexpected status code %d for gauge %s", resp.StatusCode(), name)
	}

	c.logger.Debug("gauge sent", slog.String("name", name), slog.Float64("value", value))
	return nil
}

func (c *Client) SendCounter(name string, value int64) error {
	body := m.NewCounter(name, value)
	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post("/update")

	if err != nil {
		return fmt.Errorf("failed to send counter %s: %w", name, err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("unexpected status code %d for counter %s", resp.StatusCode(), name)
	}

	c.logger.Debug("counter sent", slog.String("name", name), slog.Int64("value", value))
	return nil
}

// SendBatch отправляет пакет метрик на POST /updates/ в формате []Metrics,
// сжимая тело запроса алгоритмом gzip. Пустые батчи не отправляются.
func (c *Client) SendBatch(metrics []m.Metrics) error {
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

	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(buf.Bytes()).
		Post("/updates/")

	if err != nil {
		return fmt.Errorf("failed to send metrics batch: %w", err)
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("unexpected status code %d for metrics batch", resp.StatusCode())
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
