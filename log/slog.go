package log

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

const (
	projectName = "dbank"
)

// RequestIdKey is the key for the request ID in the context
var RequestIdKey struct{}

// HandlerRequestID is a slog.Handler that adds the request ID to the record
type HandlerRequestID struct {
	slog.Handler
}

// Handle handles the record and adds the request ID to the record
func (h HandlerRequestID) Handle(ctx context.Context, r slog.Record) error {
	if requestID, ok := ctx.Value(RequestIdKey).(string); ok {
		r.Add("request_id", slog.StringValue(requestID))
	}
	return h.Handler.Handle(ctx, r)
}

// GetLogLevel returns slog.Level by level string
func GetLogLevel(level string) slog.Level {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	return logLevel
}

// trimFuncName trims the function name from the project name
func trimFuncName(funcName string) string {
	if funcName == "" {
		return ""
	}

	parts := strings.Split(funcName, ".")
	return parts[len(parts)-1]
}

// trimFileName trims the file name from the project name
func trimFileName(fileName string) string {
	if fileName == "" {
		return ""
	}

	index := strings.LastIndex(fileName, projectName)
	if index == -1 {
		return fileName
	}

	return fileName[index+len(projectName)+1:]
}

// Replacer is a slog.Attr replacer
func Replacer(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.SourceKey {
		source := a.Value.Any().(*slog.Source)
		source.File = trimFileName(source.File)
		source.Function = trimFuncName(source.Function)
	}
	return a
}

// GetLogger returns a slog.Logger with HandlerRequestID
// and slog.Source replaced by relative path by logger level string
func GetLogger(level string) *slog.Logger {
	var options *slog.HandlerOptions
	switch level {
	case "debug":
		options = &slog.HandlerOptions{
			Level:       GetLogLevel(level),
			AddSource:   true,
			ReplaceAttr: Replacer,
		}
	default:
		options = &slog.HandlerOptions{
			Level:       GetLogLevel(level),
			AddSource:   false,
			ReplaceAttr: Replacer,
		}
	}

	handler := HandlerRequestID{Handler: slog.NewJSONHandler(os.Stderr, options)}
	return slog.New(handler).With()
}

// GetRequestID returns the request ID from the context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIdKey).(string); ok {
		return requestID
	}
	return ""
}
