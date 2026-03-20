package main

import (
	"go-zap/common"
	"go-zap/initialize"

	"go.uber.org/zap"
)

func main() {
    // 1. 初始化配置 (Viper)
    // 增加错误处理，如果配置读取失败，程序应终止
    if err := initialize.Viper(); err != nil {
        panic(err)
    }

    // 2. 初始化日志
    // 增加错误处理，如果日志初始化失败，程序应终止
    if err := initialize.Logger(); err != nil {
        panic(err)
    }

    // 3. 确保程序退出前刷新缓冲区
    // Zap 是异步日志，必须调用 Sync 才能保证所有日志写入磁盘
    defer zap.L().Sync()

    // 4. 业务逻辑
    common.Logger.Info("日志开始")
    common.Logger.Error("日志结束")
}