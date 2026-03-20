package internal

import (
	"go-zap/common"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapCore struct {
	level zapcore.Level
	zapcore.Core
}

func NewZapCore(level zapcore.Level, formats ...string) *ZapCore {
	ent := &ZapCore{level: level}

	levelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l == level
	})

	// 自定义日志切割器 Cutter,也可以使用第三方lumberjack提供的日志切割器
	cutter := NewCutter(
		common.Config.Zap.Director, level.String(), common.Config.Zap.LogFile.MaxSize,
		common.Config.Zap.LogFile.BackUps, common.Config.Zap.LogFile.Compress,
		common.Config.Zap.LogFile.MaxAge, CutterWithLayout(time.DateOnly),
		CutterWithFormats(formats...),
	)

	// 控制台输出
	consoleCore := zapcore.NewCore(common.Config.Zap.ConsoleEncoder(), zapcore.AddSync(os.Stdout), levelEnabler)
	// 写入文件
	fileCore := zapcore.NewCore(common.Config.Zap.FileEncoder(), zapcore.AddSync(cutter), levelEnabler)

	switch common.Config.Zap.Writer {
	case "console":
		ent.Core = consoleCore

	case "file":
		ent.Core = fileCore
	case "both":

		ent.Core = zapcore.NewTee(consoleCore, fileCore)

	default:
		ent.Core = fileCore
	}
	return ent
}

func (zc *ZapCore) WriteSyncer(levelEnabler zap.LevelEnablerFunc, formats ...string) zapcore.WriteSyncer {
	cutter := NewCutter(
		common.Config.Zap.Director, zc.level.String(), common.Config.Zap.LogFile.MaxSize,
		common.Config.Zap.LogFile.BackUps, common.Config.Zap.LogFile.Compress,
		common.Config.Zap.LogFile.MaxAge, CutterWithLayout(time.DateOnly),
		CutterWithFormats(formats...),
	)
	switch common.Config.Zap.Writer {
	case "console":
		return zapcore.AddSync(os.Stdout)
	case "file":
		return zapcore.AddSync(cutter)
	case "both":
		muliSyncer := zapcore.NewMultiWriteSyncer(os.Stdout, cutter)
		return zapcore.AddSync(muliSyncer)
	default:
		return zapcore.AddSync(cutter)
	}
}

func (zc *ZapCore) Enabled(level zapcore.Level) bool {
	return zc.level == level
}

func (zc *ZapCore) With(fields []zapcore.Field) zapcore.Core {
	return zc.Core.With(fields)
}

func (zc *ZapCore) Check(ent zapcore.Entry, check *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if zc.Enabled(ent.Level) {
		return check.AddCore(ent, zc)

	}
	return check
}

// Write 恢复为默认透传状态，移除了入库和错误提取逻辑
func (zc *ZapCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return zc.Core.Write(entry, fields)
}

func (zc *ZapCore) Sync() error {
	return zc.Core.Sync()
}
