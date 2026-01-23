package utils

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger() {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	encoder := zapcore.NewJSONEncoder(config)
	writer := zapcore.AddSync(os.Stdout)

	core := zapcore.NewCore(encoder, writer, zapcore.InfoLevel)

	Logger = zap.New(core, zap.AddCaller())
}

// GetLogger returns the global logger
func GetLogger() *zap.Logger {
	if Logger == nil {
		InitLogger()
	}
	return Logger
}
