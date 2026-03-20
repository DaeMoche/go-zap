package initialize

import (
	"fmt"
	"go-zap/common"
	"go-zap/initialize/internal"
	"go-zap/utils"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Logger() error {
	if ok, _ := utils.PathExists(common.Config.Zap.Director); !ok {
		fmt.Printf("Create %v directoory.\n", common.Config.Zap.Director)
		_ = os.Mkdir(common.Config.Zap.Director, os.ModePerm)
	}

	levels := common.Config.Zap.Levels()
	length := len(levels)
	cores := make([]zapcore.Core, 0, length)
	
	for i := 0; i < length; i++ {
		core := internal.NewZapCore(levels[i])
		cores = append(cores, core)
	}

	logger := zap.New(zapcore.NewTee(cores...))

	opts := []zap.Option{}

	if common.Config.Zap.Caller {
		opts = append(opts, zap.AddCaller())
	}

	if common.Config.Zap.StackTrace {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	logger = logger.WithOptions(opts...)
	zap.ReplaceGlobals(logger)
	common.Logger = logger
	return nil
}
