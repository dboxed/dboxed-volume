package util

import (
	"log/slog"
	"time"
)

func Measure(name string) func() {
	startTime := time.Now()
	return func() {
		endTime := time.Now()
		slog.Info("measure: "+name, slog.Any("time", endTime.Sub(startTime).String()))
	}
}
