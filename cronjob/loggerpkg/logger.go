// Package loggerpkg provides a shared logging utility for the application.
// It initializes a zap sugared logger with a default configuration and exposes
// methods to obtain named loggers for contextual logging.
package loggerpkg

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the main instance of the sugared logger used throughout the application.
var (
	logger      *zap.Logger
	sugarLogger *zap.SugaredLogger
)

// init as as SetupLogger - initializes the logger with a default configuration.
// This function runs when the package is imported, ensuring the logger is
// set up and ready for use in other packages.
func init() {
	runMode := os.Getenv("RUN_MODE")
	logOutput := os.Getenv("LOG_OUTPUT")
	logLevel := zap.DebugLevel // default to debug
	if runMode == "prod" {
		logLevel = zap.InfoLevel
	}

	cfg := zap.Config{
		Encoding:         logOutput, // Use console encoding
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:     "message",
			LevelKey:       "level",
			TimeKey:        "time",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    "function",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder, // Colorize levels
			EncodeTime:     zapcore.RFC3339TimeEncoder,       // Use RFC3339 time format
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		Level: zap.NewAtomicLevelAt(logLevel),
	}
	var err error
	logger, err = cfg.Build()
	if err != nil {
		panic("Failed to build logger: " + err.Error())
	}
	defer logger.Sync() // Ensure logs are flushed before exiting.
	sugarLogger = logger.Sugar()

	// for json format:
	// cfg := zap.NewProductionConfig()
	// cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// logLevel := zap.NewAtomicLevel()

	// // Here, you can set the log level to whatever you want.
	// // For instance, to set it to "debug":
	// logLevel.SetLevel(zap.DebugLevel)
	// cfg.Level = logLevel

	// logger, _ = cfg.Build()
	// defer logger.Sync() // Ensure logs are flushed before exiting.

	// sugarLogger = logger.Sugar()
}

// GetLogger returns the main instance of the sugared logger.
// This logger can be used for general logging purposes and provides methods
// like Infof, Debugf, Errorf, etc. for formatted logging.
func GetLogger() *zap.SugaredLogger {
	return sugarLogger
}

// GetNamedLogger returns a named instance of the sugared logger.
// Named loggers provide context to the logs, making it easier to trace and debug logs
// from specific parts of the application.
// For example, using GetNamedLogger("database") will prefix all logs with the context "database".
func GetNamedLogger(name string) *zap.SugaredLogger {
	return sugarLogger.Named(name)
}
