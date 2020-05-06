package main

import "go.uber.org/zap"

func init() {
	logger, _ := zap.NewDevelopment(
		zap.AddStacktrace(zap.FatalLevel),
	)
	zap.ReplaceGlobals(logger)
	defer zap.L().Sync()
}
