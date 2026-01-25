package agent

import (
	"fmt"
	"log/slog"
	"reflect"
	"strconv"

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
	resp, err := c.client.R().
		SetPathParam("name", name).
		SetPathParam("value", strconv.FormatFloat(value, 'f', -1, 64)).
		Post("/gauge/{name}/{value}")

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
	resp, err := c.client.R().
		SetPathParam("name", name).
		SetPathParam("value", strconv.FormatInt(value, 10)).
		Post("/counter/{name}/{value}")

	if err != nil {
		return fmt.Errorf("failed to send counter %s: %w", name, err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("unexpected status code %d for counter %s", resp.StatusCode(), name)
	}

	c.logger.Debug("counter sent", slog.String("name", name), slog.Int64("value", value))
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
			c.logger.Error("failed to send metric",
				slog.String("field", fieldName),
				slog.String("error", err.Error()))
		}
	}

	return nil
}
