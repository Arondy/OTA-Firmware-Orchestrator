package config

import (
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

var openFiles = make([]*lumberjack.Logger, 0, 2)
var Logger *zap.SugaredLogger = newLogger()

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
	_ = os.Truncate(LOGS_FILE, 0)
	_ = os.Truncate(TRACEBACK_FILE, 0)

	logsLogger := &lumberjack.Logger{
		Filename:   LOGS_FILE,
		MaxSize:    50,
		MaxBackups: 1,
		MaxAge:     28,
		Compress:   true,
	}
	mainWriter := zapcore.AddSync(logsLogger)

	traceLogger := &lumberjack.Logger{
		Filename:   TRACEBACK_FILE,
		MaxSize:    50,
		MaxBackups: 1,
		MaxAge:     28,
		Compress:   true,
	}
	traceWriter := zapcore.AddSync(traceLogger)

	if len(openFiles) == 0 {
		openFiles = append(openFiles, logsLogger, traceLogger)
	}

	cores := []zapcore.Core{
		zapcore.NewCore(zapcore.NewConsoleEncoder(setupEncoderConfig(true)), zapcore.AddSync(os.Stdout), zap.DebugLevel),
		zapcore.NewCore(zapcore.NewConsoleEncoder(setupEncoderConfig(false)), mainWriter, zap.InfoLevel),
		zapcore.NewCore(zapcore.NewConsoleEncoder(setupEncoderConfig(false)), traceWriter, zap.WarnLevel),
	}

	return zap.New(
		zapcore.NewTee(cores...),
		zap.AddStacktrace(zap.WarnLevel),
	).Sugar()
}

func CloseLoggerFiles() {
	for _, file := range openFiles {
		_ = file.Close()
	}
}
