package logger

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	zap *zap.Logger
}

type Config struct {
	Level      string
	Format     string
	Output     string
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
}

func New(config Config) (*Logger, error) {
	var level zapcore.Level
	switch config.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// Configure encoder
	var encoder zapcore.Encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	if config.Format == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Configure output
	var writeSyncer zapcore.WriteSyncer
	if config.Output == "file" && config.Filename != "" {
		file, err := os.OpenFile(config.Filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		writeSyncer = zapcore.AddSync(file)
	} else if config.Output == "stderr" {
		writeSyncer = zapcore.AddSync(os.Stderr)
	} else {
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(encoder, writeSyncer, level)

	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &Logger{zap: zapLogger}, nil
}

func NewDevelopment() *Logger {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	
	zapLogger, _ := config.Build()
	return &Logger{zap: zapLogger}
}

func NewProduction() *Logger {
	zapLogger, _ := zap.NewProduction()
	return &Logger{zap: zapLogger}
}

func (l *Logger) Debug(msg string, fields map[string]interface{}) {
	l.zap.Debug(msg, l.convertFields(fields)...)
}

func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.zap.Info(msg, l.convertFields(fields)...)
}

func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	l.zap.Warn(msg, l.convertFields(fields)...)
}

func (l *Logger) Error(msg string, err error, fields map[string]interface{}) {
	zapFields := l.convertFields(fields)
	if err != nil {
		zapFields = append(zapFields, zap.Error(err))
	}
	l.zap.Error(msg, zapFields...)
}

func (l *Logger) Fatal(msg string, err error, fields map[string]interface{}) {
	zapFields := l.convertFields(fields)
	if err != nil {
		zapFields = append(zapFields, zap.Error(err))
	}
	l.zap.Fatal(msg, zapFields...)
}

func (l *Logger) Sync() error {
	return l.zap.Sync()
}

func (l *Logger) convertFields(fields map[string]interface{}) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	
	for key, value := range fields {
		switch v := value.(type) {
		case string:
			zapFields = append(zapFields, zap.String(key, v))
		case int:
			zapFields = append(zapFields, zap.Int(key, v))
		case int64:
			zapFields = append(zapFields, zap.Int64(key, v))
		case float64:
			zapFields = append(zapFields, zap.Float64(key, v))
		case bool:
			zapFields = append(zapFields, zap.Bool(key, v))
		case time.Time:
			zapFields = append(zapFields, zap.Time(key, v))
		case time.Duration:
			zapFields = append(zapFields, zap.Duration(key, v))
		case error:
			zapFields = append(zapFields, zap.Error(v))
		case []byte:
			zapFields = append(zapFields, zap.Binary(key, v))
		default:
			zapFields = append(zapFields, zap.Any(key, v))
		}
	}
	
	return zapFields
}

func (l *Logger) With(fields map[string]interface{}) *Logger {
	return &Logger{zap: l.zap.With(l.convertFields(fields)...)}
}


func (l *Logger) Named(name string) *Logger {
	return &Logger{zap: l.zap.Named(name)}
}

func (l *Logger) GetZapLogger() *zap.Logger {
	return l.zap
}
