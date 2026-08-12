package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigNormalized(t *testing.T) {
	def := DefaultConfig()

	tests := []struct {
		name string
		in   *Config
		want Config
	}{
		{
			name: "nil gives defaults",
			in:   nil,
			want: *def,
		},
		{
			name: "zero durations mean unlimited, zero counters get defaults",
			in:   &Config{},
			want: Config{MaxOpenConns: def.MaxOpenConns, MaxIdleConns: def.MaxIdleConns, PingTimeout: def.PingTimeout},
		},
		{
			name: "idle clamped to open",
			in:   &Config{MaxOpenConns: 3, MaxIdleConns: 20, ConnMaxIdleTime: time.Second, ConnMaxLifetime: time.Minute, PingTimeout: time.Second},
			want: Config{MaxOpenConns: 3, MaxIdleConns: 3, ConnMaxIdleTime: time.Second, ConnMaxLifetime: time.Minute, PingTimeout: time.Second},
		},
		{
			name: "negative durations mean unlimited",
			in:   &Config{MaxOpenConns: 50, MaxIdleConns: 25, ConnMaxIdleTime: -time.Second, ConnMaxLifetime: -time.Second, PingTimeout: -time.Second},
			want: Config{MaxOpenConns: 50, MaxIdleConns: 25, ConnMaxIdleTime: 0, ConnMaxLifetime: 0, PingTimeout: def.PingTimeout},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.in.normalized())
		})
	}
}
