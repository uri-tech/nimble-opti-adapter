// Package loggerpkg provides a shared logging utility for the application.
// It initializes a zap sugared logger with a default configuration and exposes
// methods to obtain named loggers for contextual logging.
package loggerpkg

import (
	"os"
	"strconv"
	"strings"

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
	// Set default log level to debug.
	logLevel := zap.DebugLevel
	disableCaller := false
	if runMode := os.Getenv("RUN_MODE"); runMode != "dev" {
		logLevel = zap.InfoLevel
		disableCaller = true
	}

	// Set default log output to console.
	logOutput := "console"
	if envLogOutput := os.Getenv("LOG_OUTPUT"); envLogOutput != "" {
		logOutput = envLogOutput
	}

	// Initialize logger with default configuration.
	cfg := zap.Config{
		Development:       !disableCaller,
		DisableStacktrace: disableCaller,
		Encoding:          logOutput, // Use console encoding
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		DisableCaller:     disableCaller,
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
			// EncodeCaller: customCallerEncoder,
		},
		Level: zap.NewAtomicLevelAt(logLevel),
	}
	var err error
	logger, err = cfg.Build()
	// logger, err = cfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic("Failed to build logger: " + err.Error())
	}
	// defer logger.Sync() // Ensure logs are flushed before exiting.
	sugarLogger = logger.Sugar()
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

// customCallerEncoder is a custom function to encode the caller information in a concise format.
// Instead of displaying the full path, it shows only the folder and file name, followed by the line number.
// For example, it transforms:
// "github.com/uri-tech/nimble-opti-adapter/cronjob/internal/ingresswatcher/challenge.go:51"
// into:
// "ingresswatcher/challenge.go:51"
//
// Parameters:
// - caller: Contains information about the calling context, including the file and line number.
// - enc: An encoder to which the formatted caller information is appended.
func customCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	// If the caller information is not defined, append "undefined" to the log entry.
	if !caller.Defined {
		enc.AppendString("undefined")
		return
	}

	// Split the full file path into segments.
	segments := strings.Split(caller.FullPath(), "/")

	// If there are more than one segments, take the last two segments (folder and file name),
	// and append them along with the line number to the log entry.
	if len(segments) > 1 {
		enc.AppendString(segments[len(segments)-2] + "/" + segments[len(segments)-1] + ":" + strconv.Itoa(caller.Line))
	} else {
		// If there's only one segment (or none), append the full path and line number to the log entry.
		enc.AppendString(caller.FullPath() + ":" + strconv.Itoa(caller.Line))
	}
}
