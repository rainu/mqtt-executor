package main

import "go.uber.org/zap"

func init() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
	defer zap.L().Sync()
}
