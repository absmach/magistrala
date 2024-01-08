package logger

import (
	"fmt"
	"io"
	"time"
	"context"

	"log/slog"
)

// Logger specifies logging API.
type Logger interface {
	Debug(context.Context, string)
	Info(context.Context, string)
	Warn(context.Context, string)
	Error(context.Context, string)
}

type logger struct {
	slogLogger slog.Logger
	level      Level
}

// New returns a new slog logger.
func New(w io.Writer, levelText string) (Logger, error) {
    var level Level
err := level.UnmarshalText(levelText)
if err != nil {
	return nil, fmt.Errorf(`{"level":"error","message":"%s: %s","ts":"%s"}`, err, levelText, time.RFC3339Nano)
}

logHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
	Level:     slog.Level(level),
	AddSource: true,
})

slogLogger := slog.New(logHandler)

return &logger{*slogLogger, level}, nil
}

func (l *logger) Debug(ctx context.Context, msg string) {
    if  Debug.isAllowed(l.level){
        l.slogLogger.Log(ctx, slog.LevelDebug, msg)
    }
}

func (l *logger) Info(ctx context.Context, msg string) {
    if Info.isAllowed(l.level) {
        l.slogLogger.Log(ctx, slog.LevelInfo, msg)
    }
}

func (l *logger) Warn(ctx context.Context, msg string) {
    if Warn.isAllowed(l.level) {
        l.slogLogger.Log(ctx, slog.LevelWarn, msg)
    }
}

func (l *logger) Error(ctx context.Context, msg string) {
    if Warn.isAllowed(l.level) {
        l.slogLogger.Log(ctx, slog.LevelError, msg)
    }
}
