package slogstyler

import "log/slog"

const DefaultTimeFormat = "2006-01-02 15:04:05.000"

type Options struct {
	Level      slog.Level
	AddSource  bool
	Colorful   bool
	Multiline  bool
	TimeFormat string
}

func DefaultOptions() *Options {
	return &Options{
		Level:      slog.LevelInfo,
		Colorful:   true,
		TimeFormat: DefaultTimeFormat,
	}
}
