package slogpretty

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	stdLog "log"
	"log/slog"
	"strings"
	"time"

	color "github.com/fatih/color"
)

type PrettyHandlerOptions struct {
	SlogOpts *slog.HandlerOptions
}

type PrettyHandler struct {
	opts PrettyHandlerOptions
	slog.Handler
	l     *stdLog.Logger
	attrs []slog.Attr
}

func (opts PrettyHandlerOptions) NewPrettyHandler(
	out io.Writer,
) *PrettyHandler {
	h := &PrettyHandler{
		Handler: slog.NewJSONHandler(out, opts.SlogOpts),
		l:       stdLog.New(out, "", 0),
	}

	return h
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	var levelColor *color.Color
	// Add color for text log message.
	switch r.Level {
	case slog.LevelDebug:
		levelColor = color.New(color.BgYellow)
	case slog.LevelError:
		levelColor = color.New(color.BgRed)
	case slog.LevelWarn:
		levelColor = color.New(color.BgGreen)
	case slog.LevelInfo:
		levelColor = color.New(color.FgBlue)
	default:
		levelColor = color.New(color.FgHiWhite)
	}

	b.WriteString(fmt.Sprintf("[%s]", r.Level.String()))
	b.WriteString(fmt.Sprintf(" %s: ", r.Time.Format(time.RFC3339)))
	b.WriteString(r.Message)

	var jsonData string
	r.Attrs(func(attr slog.Attr) bool {
		// Try to parse any attribute value as JSON
		var parsedData map[string]interface{}
		if err := json.Unmarshal([]byte(attr.Value.String()), &parsedData); err == nil {
			// Format valid JSON with indentation
			formattedJSON, _ := json.MarshalIndent(parsedData, "", "  ")
			b.WriteString(fmt.Sprintf("%s=\n%s ", attr.Key, string(formattedJSON)))
		} else {
			// Fallback to normal key=value format
			b.WriteString(fmt.Sprintf("%s=%v ", attr.Key, attr.Value))
		}

		return true
	})

	// Check "" and make struct `map`.
	if jsonData != "" {
		var parsedData map[string]interface{}
		if err := json.Unmarshal([]byte(jsonData), &parsedData); err != nil {
			h.l.Println(levelColor.Sprint(b.String())) // Return text log, if JSON != JSON.
			return err
		}
		// New format for JSON text.
		formattedJSON, err := json.MarshalIndent(parsedData, "", "  ")
		if err != nil {
			h.l.Println(levelColor.Sprintf("Failed to format JSON: %v", err))
			return nil
		}
		// Add formated JSON to text log.
		b.Write(formattedJSON)
	}
	// All log`s have color.
	h.l.Println(levelColor.Sprint(b.String()))
	return nil
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.attrs = append(h.attrs, attrs...)
	return &PrettyHandler{
		opts:    h.opts,
		Handler: h.Handler.WithAttrs(attrs),
		l:       h.l,
		attrs:   h.attrs,
	}
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return &PrettyHandler{
		opts:    h.opts,
		Handler: h.Handler.WithGroup(name),
		l:       h.l,
		attrs:   h.attrs,
	}
}
