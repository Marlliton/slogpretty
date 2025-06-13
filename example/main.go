package main

import (
	"log/slog"
	"os"

	"github.com/Marlliton/slogpretty"
)

func main() {
	handler := slogpretty.New(os.Stdout, &slogpretty.Options{
		Level:      slog.LevelDebug,
		Colorful:   true,
		AddSource:  true,
		Multiline:  true,
		TimeFormat: slogpretty.DefaultTimeFormat,
	})
	l := slog.New(handler)
	slog.SetDefault(l)

	slog.Info("Event with group and subgroups",
		"user", "bob",
		slog.Group("details",
			slog.Int("port", 8080),
			slog.String("status", "inactive"),
			slog.Group("metrics",
				slog.Float64("cpu", 72.5),
				slog.Float64("memory", 65.3),
			),
			slog.Group("location",
				slog.String("country", "Brazil"),
				slog.String("region", "SP"),
				slog.Group("coordinates",
					slog.Float64("lat", -23.5505),
					slog.Float64("lon", -46.6333),
				),
			),
		),
		"session", "0x93AF21",
		"authenticated", false,
	)
}
