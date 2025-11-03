package utils

import (
	"io"
	"os"
	"path/filepath"

	"log/slog"

	"github.com/fatih/color"
	"github.com/m-mizutani/clog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/masq"
	"github.com/m-mizutani/octovy/pkg/domain/types"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

func init() {
	_ = ReconfigureLogger("text", "info", "stdout")
}

func Logger() *slog.Logger {
	return logger
}

func ReconfigureLogger(logFormat, logLevel, logOutput string) error {
	filter := masq.New(
		// Mask value with `masq:"secret"` tag
		masq.WithTag("secret"),
		masq.WithType[types.GitHubAppSecret](masq.MaskWithSymbol('*', 64)),
		masq.WithType[types.GitHubAppPrivateKey](masq.MaskWithSymbol('*', 16)),
	)

	levelMap := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}

	level, ok := levelMap[logLevel]
	if !ok {
		return goerr.Wrap(types.ErrInvalidOption, "invalid log level", goerr.V("value", logLevel))
	}

	var w io.Writer
	switch logOutput {
	case "stdout", "-":
		w = os.Stdout
	case "stderr":
		w = os.Stderr
	default:
		fd, err := os.Create(filepath.Clean(logOutput))
		if err != nil {
			return goerr.Wrap(err, "failed to open log file", goerr.V("path", logOutput))
		}
		w = fd
	}

	var handler slog.Handler
	switch logFormat {
	case "text":
		handler = clog.New(
			clog.WithWriter(w),
			clog.WithLevel(level),
			// clog.WithReplaceAttr(filter),
			clog.WithSource(true),
			// clog.WithTimeFmt("2006-01-02 15:04:05"),
			clog.WithColorMap(&clog.ColorMap{
				Level: map[slog.Level]*color.Color{
					slog.LevelDebug: color.New(color.FgGreen, color.Bold),
					slog.LevelInfo:  color.New(color.FgCyan, color.Bold),
					slog.LevelWarn:  color.New(color.FgYellow, color.Bold),
					slog.LevelError: color.New(color.FgRed, color.Bold),
				},
				LevelDefault: color.New(color.FgBlue, color.Bold),
				Time:         color.New(color.FgWhite),
				Message:      color.New(color.FgHiWhite),
				AttrKey:      color.New(color.FgHiCyan),
				AttrValue:    color.New(color.FgHiWhite),
			}),
			clog.WithReplaceAttr(filter),
		)

	case "json":
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			AddSource:   true,
			Level:       level,
			ReplaceAttr: filter,
		})

	default:
		return goerr.Wrap(types.ErrInvalidOption, "invalid log format, should be 'json' or 'text'", goerr.V("value", logFormat))
	}

	logger = slog.New(handler)

	return nil
}
