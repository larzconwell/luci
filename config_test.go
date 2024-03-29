package luci

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Config{
		Address:           ":http",
		RouteTimeout:      time.Second,
		ReadHeaderTimeout: time.Second,
		ShutdownTimeout:   5 * time.Second,
		Logger:            DefaultConfig.Logger,
	}, DefaultConfig)
	assert.NotNil(t, DefaultConfig.Logger)
}

func TestBuildConfig(t *testing.T) {
	t.Parallel()

	t.Run("builds config from default config", func(t *testing.T) {
		t.Parallel()

		config := buildConfig(Config{})

		assert.Equal(t, DefaultConfig, config)
	})

	t.Run("nonzero config fields override default config", func(t *testing.T) {
		t.Parallel()

		config := buildConfig(Config{Address: ":0"})
		assert.Equal(t, Config{
			Address:           ":0",
			RouteTimeout:      DefaultConfig.RouteTimeout,
			ReadHeaderTimeout: DefaultConfig.ReadHeaderTimeout,
			ShutdownTimeout:   DefaultConfig.ShutdownTimeout,
			Logger:            DefaultConfig.Logger,
		}, config)

		config = buildConfig(Config{RouteTimeout: time.Hour})
		assert.Equal(t, Config{
			Address:           DefaultConfig.Address,
			RouteTimeout:      time.Hour,
			ReadHeaderTimeout: DefaultConfig.ReadHeaderTimeout,
			ShutdownTimeout:   DefaultConfig.ShutdownTimeout,
			Logger:            DefaultConfig.Logger,
		}, config)

		config = buildConfig(Config{ReadHeaderTimeout: time.Hour})
		assert.Equal(t, Config{
			Address:           DefaultConfig.Address,
			RouteTimeout:      DefaultConfig.RouteTimeout,
			ReadHeaderTimeout: time.Hour,
			ShutdownTimeout:   DefaultConfig.ShutdownTimeout,
			Logger:            DefaultConfig.Logger,
		}, config)

		config = buildConfig(Config{ShutdownTimeout: time.Hour})
		assert.Equal(t, Config{
			Address:           DefaultConfig.Address,
			RouteTimeout:      DefaultConfig.RouteTimeout,
			ReadHeaderTimeout: DefaultConfig.ReadHeaderTimeout,
			ShutdownTimeout:   time.Hour,
			Logger:            DefaultConfig.Logger,
		}, config)

		config = buildConfig(Config{Logger: noopLogger})
		assert.Equal(t, Config{
			Address:           DefaultConfig.Address,
			RouteTimeout:      DefaultConfig.RouteTimeout,
			ReadHeaderTimeout: DefaultConfig.ReadHeaderTimeout,
			ShutdownTimeout:   DefaultConfig.ShutdownTimeout,
			Logger:            noopLogger,
		}, config)
	})
}
