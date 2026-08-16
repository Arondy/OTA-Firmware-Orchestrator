package config

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	colorTime      = "\x1b[38;2;84;139;42m"
	colorClass     = "\x1b[38;2;128;128;128m"
	colorReset     = "\x1b[0m"
	LOGS_FILE      = "logs.log"
	TRACEBACK_FILE = "traceback.log"
)

var logger *zap.SugaredLogger = newLogger()
var logsFileWriter *lumberjack.Logger
var traceFileWriter *lumberjack.Logger

func setupEncoderConfig(useColors bool) zapcore.EncoderConfig {
	cfg := zapcore.EncoderConfig{
		MessageKey:       "M",
		TimeKey:          "T",
		LevelKey:         "L",
		NameKey:          "N",
		StacktraceKey:    "S",
		ConsoleSeparator: " | ",
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			tStr := t.Format("02-01 15:04:05")
			if useColors {
				tStr = colorTime + tStr + colorReset
			}
			enc.AppendString(tStr)
		},
		EncodeName: func(name string, enc zapcore.PrimitiveArrayEncoder) {
			if useColors {
				name = colorClass + name + colorReset
			}
			enc.AppendString(name)
		},
	}
	if useColors {
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	return cfg
}

func newLogger() *zap.SugaredLogger {
	logsFileWriter = &lumberjack.Logger{
		Filename:   LOGS_FILE,
		MaxSize:    50,
		MaxBackups: 1,
		MaxAge:     28,
		Compress:   true,
	}

	traceFileWriter = &lumberjack.Logger{
		Filename:   TRACEBACK_FILE,
		MaxSize:    50,
		MaxBackups: 1,
		MaxAge:     28,
		Compress:   true,
	}

	cores := []zapcore.Core{
		zapcore.NewCore(zapcore.NewConsoleEncoder(setupEncoderConfig(true)), zapcore.AddSync(os.Stdout), zap.DebugLevel),
		zapcore.NewCore(zapcore.NewJSONEncoder(setupEncoderConfig(false)),  zapcore.AddSync(logsFileWriter), zap.InfoLevel),
		zapcore.NewCore(zapcore.NewJSONEncoder(setupEncoderConfig(false)), zapcore.AddSync(traceFileWriter), zap.WarnLevel),
	}

	return zap.New(
		zapcore.NewTee(cores...),
		zap.AddStacktrace(zap.ErrorLevel),
	).Sugar()
}

func GetLogger() *zap.SugaredLogger {
	return logger
}

func CloseLoggerFiles() {
    if logsFileWriter != nil {
        _ = logsFileWriter.Close()
    }
    if traceFileWriter != nil {
        _ = traceFileWriter.Close()
    }
}

type loggerKey struct{}

func LoggerToContext(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func LoggerFromContext(ctx context.Context) *zap.SugaredLogger {
	ctxLogger, ok := ctx.Value(loggerKey{}).(*zap.SugaredLogger)
	if !ok {
		return logger
	}
	return ctxLogger
}
