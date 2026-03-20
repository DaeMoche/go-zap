package config

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type ZapConfig struct {
	Prefix      string         `yaml:"prefix" mapstructure:"prefix"`
	TimeFormat  string         `yaml:"timeFormat" mapstructure:"timeFormat"` // 支持 "iso8601", "rfc3339", "millis", "nanos"
	Level       string         `yaml:"level" mapstructure:"level"`           // debug, info, warn, error, dpanic, panic, fatal
	Caller      bool           `yaml:"caller" mapstructure:"caller"`
	StackTrace  bool           `yaml:"stackTrace" mapstructure:"stackTrace"`
	Writer      string         `yaml:"writer" mapstructure:"writer"`             // console, file, both
	Encode      string         `yaml:"encode" mapstructure:"encode"`             // json, console
	EncodeLevel string         `yaml:"encode-level" mapstructure:"encode-level"` // json, console
	Director    string         `yaml:"director" mapstructure:"director"`
	LogFile     *LogFileConfig `yaml:"logFile" mapstructure:"logFile"`
}

type LogFileConfig struct {
	MaxAge   int  `yaml:"maxAge" mapstructure:"maxAge"`
	MaxSize  int  `yaml:"maxSize" mapstructure:"maxSize"`
	BackUps  int  `yaml:"backups" mapstructure:"backups"`
	Compress bool `yaml:"compress" mapstructure:"compress"`
}

type prefixEncoder struct {
	zapcore.Encoder
	prefix string
}

// GetLevel 将字符串级别转换为 zapcore.Level
func (z *ZapConfig) Levels() []zapcore.Level {
	levels := make([]zapcore.Level, 0, 7)
	level, err := zapcore.ParseLevel(z.Level)
	if err != nil {
		level = zapcore.DebugLevel
	}
	for ; level <= zap.FatalLevel; level++ {
		levels = append(levels, level)
	}
	return levels
}

// customEncoder 根据 TimeFormat 返回对应的时间编码器
func (z *ZapConfig) timeEncoder(t time.Time, encoder zapcore.PrimitiveArrayEncoder) {
	encoder.AppendString(t.Format(z.TimeFormat))
}

// plainLevelEncoder 返回不带颜色的 LevelEncoder (用于文件，强制去除颜色)
func (z *ZapConfig) plainLevelEncoder() zapcore.LevelEncoder {
	switch {
	case z.EncodeLevel == "LowercaseColorLevelEncoder", z.EncodeLevel == "LowercaseLevelEncoder":
		return zapcore.LowercaseLevelEncoder
	case z.EncodeLevel == "CapitalColorLevelEncoder", z.EncodeLevel == "CapitalLevelEncoder":
		return zapcore.CapitalLevelEncoder
	default:
		// 默认返回大写不带颜色
		return zapcore.CapitalLevelEncoder
	}
}

func (z *ZapConfig) levelEncoder() zapcore.LevelEncoder {
	switch {
	case z.EncodeLevel == "LowercaseColorLevelEncoder":
		return zapcore.LowercaseColorLevelEncoder
	case z.EncodeLevel == "LowercaseLevelEncoder":
		return zapcore.LowercaseLevelEncoder
	case z.EncodeLevel == "CapitalLevelEncoder":
		return zapcore.CapitalLevelEncoder
	case z.EncodeLevel == "CapitalColorLevelEncoder":
		return zapcore.CapitalColorLevelEncoder
	default:
		return zapcore.LowercaseColorLevelEncoder
	}
}

// buildEncoderConfig 创建基础的 EncoderConfig
func (z *ZapConfig) buildEncoderConfig(levelEncoder zapcore.LevelEncoder) zapcore.EncoderConfig {

	return zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     z.timeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeLevel:    levelEncoder, // 使用传入的 levelEncoder
	}

}

// ConsoleEncoder 创建控制台用的 Encoder (支持颜色)
func (z *ZapConfig) ConsoleEncoder() zapcore.Encoder {
	// 控制台使用 levelEncoder (允许颜色)
	encoderConfig := z.buildEncoderConfig(z.levelEncoder())

	var encoder zapcore.Encoder
	if z.Encode == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	if z.Prefix != "" {
		return &prefixEncoder{
			Encoder: encoder,
			prefix:  z.Prefix,
		}
	}

	return encoder
}

// FileEncoder 创建文件用的 Encoder (强制无颜色)
func (z *ZapConfig) FileEncoder() zapcore.Encoder {
	// 文件使用 plainLevelEncoder (强制无颜色)
	encoderConfig := z.buildEncoderConfig(z.plainLevelEncoder())

	var encoder zapcore.Encoder
	if z.Encode == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	if z.Prefix != "" {
		return &prefixEncoder{
			Encoder: encoder,
			prefix:  z.Prefix,
		}
	}

	return encoder
}

// EncodeEntry 重写编码方法，先写入前缀，再写入原始日志内容
func (e *prefixEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {

	// 2. 再调用底层 Encoder 写入剩余内容
	buffer, err := e.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return nil, err
	}
	logLine := buffer.String()
	buffer.Reset()
	buffer.AppendString(e.prefix + logLine)
	return buffer, nil
}

func (e *prefixEncoder) Clone() zapcore.Encoder {
	return &prefixEncoder{
		Encoder: e.Encoder.Clone(),
		prefix:  e.prefix,
	}
}
