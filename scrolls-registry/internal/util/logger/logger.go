package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
)

const (
	LogKeyContext    = "context"
	LogContextUpdate = "update"
	LogContextView   = "view"
	LogContextPush   = "push"
)

var Log *zap.Logger

func init() {
	Log = NewLogger("main")
	Log.Info("Initializing logger...")
}

func NewLogger(name string) *zap.Logger {
	encoder := zap.NewDevelopmentEncoderConfig()
	encoder.ConsoleSeparator = " "
	encoder.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString("[" + l.CapitalString() + "]")
	}
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:      true,
		DisableCaller:    true,
		Encoding:         "console",
		EncoderConfig:    encoder,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	l, err := cfg.Build()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	if os.Getenv("APP_ENV") != "live" {
		l.Sugar()
	}
	l.Named(name)
	return l
}
