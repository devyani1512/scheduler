package main

import "go.uber.org/zap"

func buildLogger(level string) *zap.Logger {
	if level == "debug" {
		l, _ := zap.NewDevelopment()
		return l
	}
	l, _ := zap.NewProduction()
	return l
}
