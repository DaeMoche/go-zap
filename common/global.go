package common

import (
	"go-zap/config"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	Config config.Config
	Viper  *viper.Viper
	Logger *zap.Logger
)
