package proto

import (
	"testing"

	"github.com/d2cTool/rtmetrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromModelToModelRoundTrip(t *testing.T) {
	t.Parallel()

	delta := int64(7)
	value := 1.5
	src := []model.Metrics{
		{ID: "PollCount", MType: model.Counter, Delta: &delta},
		{ID: "Alloc", MType: model.Gauge, Value: &value},
	}

	got := ToModel(FromModel(src))
	require.Len(t, got, 2)
	assert.Equal(t, "PollCount", got[0].ID)
	assert.Equal(t, model.Counter, got[0].MType)
	require.NotNil(t, got[0].Delta)
	assert.Equal(t, int64(7), *got[0].Delta)
	assert.Equal(t, "Alloc", got[1].ID)
	assert.Equal(t, model.Gauge, got[1].MType)
	require.NotNil(t, got[1].Value)
	assert.Equal(t, 1.5, *got[1].Value)
}

func TestToModelSkipsEmptyAndUnknown(t *testing.T) {
	t.Parallel()

	got := ToModel([]*Metric{
		nil,
		{Id: "", Type: Metric_GAUGE, Value: 1},
		{Id: "x", Type: Metric_MType(99)},
		{Id: "Alloc", Type: Metric_GAUGE, Value: 2},
	})
	require.Len(t, got, 1)
	assert.Equal(t, "Alloc", got[0].ID)
}
