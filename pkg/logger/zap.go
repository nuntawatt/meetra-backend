// Package logger provides a singleton zap logger for the application.
package logger

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log  *zap.Logger
	once sync.Once
)

// Init initialises the global logger. Safe to call only once.
// env should be "development" or "production".
func Init(env string) {
	once.Do(func() {
		var cfg zap.Config
		if env == "production" {
			cfg = zap.NewProductionConfig()
		} else {
			cfg = zap.NewDevelopmentConfig()
			cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}

		var err error
		log, err = cfg.Build()
		if err != nil {
			panic("failed to initialise logger: " + err.Error())
		}
	})
}

// Get returns the singleton zap logger. Panics if Init was not called.
func Get() *zap.Logger {
	if log == nil {
		panic("logger not initialised: call logger.Init first")
	}
	return log
}

// Sync flushes any buffered log entries. Call this before process exit.
func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}
