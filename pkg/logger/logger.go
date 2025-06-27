package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	ServiceName string `yaml:"service_name" env:"LOGGER_SERVICE_NAME" env-default:"xengate" env-description:"Service name"`
	Level       string `yaml:"level" env:"LOGGER_LEVEL" env-default:"debug" env-description:"Enabled verbose logging"`
	Pretty      bool   `yaml:"pretty" env:"LOGGER_PRETTY" env-default:"false" env-description:"Enables human readable logging. Otherwise, uses json output"`
}

func New(cfg Config) *zap.SugaredLogger {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.DebugLevel
	}

	atomicLevel := zap.NewAtomicLevelAt(level)

	encoder := getEncoder()

	fileWriter := getLogWriter(cfg.ServiceName)

	fileCore := zapcore.NewCore(encoder, fileWriter, atomicLevel)

	consoleWriter := zapcore.Lock(os.Stdout)
	consoleCore := zapcore.NewCore(encoder, consoleWriter, atomicLevel)

	core := zapcore.NewTee(fileCore, consoleCore)

	logger := zap.New(core, zap.AddCaller()).Sugar()

	return logger
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		// EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeLevel: CustomLevelEncoder,
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter(serviceName string) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "logs/" + serviceName + `.log`,
		MaxSize:    200, // MB
		MaxBackups: 30,
		MaxAge:     90, // days
		Compress:   true,
	}
	return zapcore.AddSync(lumberJackLogger)
}

func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + getIcon(level) + level.CapitalString() + "]")
}

func getIcon(lvl zapcore.Level) string {
	switch lvl {
	case zapcore.InfoLevel:
		return "🔵 "
	case zapcore.DebugLevel:
		return "🟢 "
	case zapcore.WarnLevel:
		return "🟡️ "
	case zapcore.ErrorLevel:
		return "🔴 "
	case zapcore.FatalLevel, zapcore.PanicLevel:
		return "⚫ "
	default:
		return ""
	}
}
