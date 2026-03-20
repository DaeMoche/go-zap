package config

import (
	"sync/atomic"
)

type Config struct {
	Zap ZapConfig `mapstructure:"zap" yaml:"zap" `
}

var (
	ConfigType       string = "VIPER"
	TestConfigure    string = "test.config.yaml"
	DebugConfigure   string = "debug.config.yaml"
	ReleaseConfigure string = "release.config.yaml"
	DefaultConfigure string = "default.config.yaml"
	mode             atomic.Value
)

const (
	DebugMode   = "debug"
	TestMode    = "test"
	ReleaseMode = "release"
)

func Mode() string {
	return mode.Load().(string)
}
