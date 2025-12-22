// Copyright (C) 2024-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel represents logging levels
type LogLevel string

const (
	LogLevelVerbo LogLevel = "verbo"
	LogLevelDebug LogLevel = "debug"
	LogLevelTrace LogLevel = "trace"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
	LogLevelOff   LogLevel = "off"
)

// LogFormat represents log output formats
type LogFormat string

const (
	LogFormatTerminal LogFormat = "terminal"
	LogFormatJSON     LogFormat = "json"
	LogFormatPlain    LogFormat = "plain"
)

// LogFactory creates configured loggers
type LogFactory struct {
	config LogConfig
}

// NewLogFactory creates a new log factory from configuration
func NewLogFactory(cfg LogConfig) *LogFactory {
	return &LogFactory{config: cfg}
}

// NewLogFactoryFromGlobal creates a log factory from global config
func NewLogFactoryFromGlobal() *LogFactory {
	return NewLogFactory(Global().Log)
}

// CreateLogger creates a new logger with the given name
func (f *LogFactory) CreateLogger(name string) (*zap.Logger, error) {
	// Parse level
	level := f.parseLevel()

	// Create encoder config
	encoderConfig := f.encoderConfig()

	// Create encoder based on format
	encoder := f.createEncoder(encoderConfig)

	// Create outputs
	cores := f.createCores(encoder, encoderConfig, level, name)

	// Combine cores
	core := zapcore.NewTee(cores...)

	// Build logger options
	opts := []zap.Option{
		zap.AddStacktrace(zapcore.ErrorLevel),
	}
	if f.config.ShowCaller {
		opts = append(opts, zap.AddCaller())
	}

	return zap.New(core, opts...).Named(name), nil
}

// parseLevel converts string level to zapcore.Level
func (f *LogFactory) parseLevel() zapcore.Level {
	switch LogLevel(f.config.Level) {
	case LogLevelVerbo, LogLevelTrace, LogLevelDebug:
		return zapcore.DebugLevel
	case LogLevelInfo:
		return zapcore.InfoLevel
	case LogLevelWarn:
		return zapcore.WarnLevel
	case LogLevelError:
		return zapcore.ErrorLevel
	case LogLevelFatal:
		return zapcore.FatalLevel
	case LogLevelOff:
		return zapcore.FatalLevel + 1 // Effectively disables logging
	default:
		return zapcore.InfoLevel
	}
}

// encoderConfig creates the encoder configuration
func (f *LogFactory) encoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    f.levelEncoder(),
		EncodeTime:     f.timeEncoder(),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// createEncoder creates the appropriate encoder based on format
func (f *LogFactory) createEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	switch LogFormat(f.config.Format) {
	case LogFormatJSON:
		return zapcore.NewJSONEncoder(cfg)
	case LogFormatPlain:
		return zapcore.NewConsoleEncoder(cfg)
	default: // terminal
		if f.config.ShowColors {
			cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
		return zapcore.NewConsoleEncoder(cfg)
	}
}

// createCores creates the logging cores (console and file)
func (f *LogFactory) createCores(encoder zapcore.Encoder, cfg zapcore.EncoderConfig, level zapcore.Level, name string) []zapcore.Core {
	var cores []zapcore.Core

	// Console output
	consoleCore := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)
	cores = append(cores, consoleCore)

	// File output (if directory specified and not empty)
	if f.config.Directory != "" {
		if err := os.MkdirAll(f.config.Directory, 0755); err == nil {
			logPath := filepath.Join(f.config.Directory, name+".log")
			fileWriter := &lumberjack.Logger{
				Filename:   logPath,
				MaxSize:    f.config.MaxSize,
				MaxBackups: f.config.MaxFiles,
				MaxAge:     f.config.MaxAge,
				Compress:   f.config.Compress,
			}

			// Always use JSON for files (easier to parse)
			fileEncoder := zapcore.NewJSONEncoder(cfg)
			fileCore := zapcore.NewCore(
				fileEncoder,
				zapcore.AddSync(fileWriter),
				level,
			)
			cores = append(cores, fileCore)
		}
	}

	return cores
}

// levelEncoder returns the level encoder based on configuration
func (f *LogFactory) levelEncoder() zapcore.LevelEncoder {
	if f.config.ShowColors && f.config.Format == string(LogFormatTerminal) {
		return zapcore.CapitalColorLevelEncoder
	}
	return zapcore.CapitalLevelEncoder
}

// timeEncoder returns the time encoder
func (f *LogFactory) timeEncoder() zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02T15:04:05.000Z07:00"))
	}
}

// CreateNopLogger creates a no-op logger that discards all output
func CreateNopLogger() *zap.Logger {
	return zap.NewNop()
}

// CreateDevelopmentLogger creates a logger suitable for development
func CreateDevelopmentLogger(name string) (*zap.Logger, error) {
	config := LogConfig{
		Level:      "debug",
		Format:     "terminal",
		ShowCaller: true,
		ShowColors: true,
	}
	return NewLogFactory(config).CreateLogger(name)
}

// CreateProductionLogger creates a logger suitable for production
func CreateProductionLogger(name string, logDir string) (*zap.Logger, error) {
	config := LogConfig{
		Level:      "info",
		Format:     "json",
		Directory:  logDir,
		MaxSize:    100,
		MaxFiles:   10,
		MaxAge:     30,
		Compress:   true,
		ShowCaller: true,
		ShowColors: false,
	}
	return NewLogFactory(config).CreateLogger(name)
}

// LoggerAdapter wraps zap.Logger to provide additional functionality
type LoggerAdapter struct {
	*zap.Logger
	sugared *zap.SugaredLogger
}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter(logger *zap.Logger) *LoggerAdapter {
	return &LoggerAdapter{
		Logger:  logger,
		sugared: logger.Sugar(),
	}
}

// Sugared returns the sugared logger for printf-style logging
func (l *LoggerAdapter) Sugared() *zap.SugaredLogger {
	return l.sugared
}

// WithContext adds common context fields
func (l *LoggerAdapter) WithContext(ctx map[string]interface{}) *LoggerAdapter {
	fields := make([]zap.Field, 0, len(ctx))
	for k, v := range ctx {
		fields = append(fields, zap.Any(k, v))
	}
	return NewLoggerAdapter(l.Logger.With(fields...))
}

// FormatError provides consistent error formatting
// This fixes issues like "last checked %s" format bugs
func FormatError(err error, context string, args ...interface{}) string {
	if len(args) > 0 {
		context = fmt.Sprintf(context, args...)
	}
	if err != nil {
		return fmt.Sprintf("%s: %v", context, err)
	}
	return context
}
