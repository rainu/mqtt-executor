package main

import (
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

var atomicLevel zap.AtomicLevel

func init() {
	//initialise our global logger
	atomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel)

	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.Level = atomicLevel
	logger, _ := zapConfig.Build(
		zap.AddStacktrace(zap.FatalLevel),
	)

	zap.ReplaceGlobals(logger)

	MQTT.ERROR, _ = zap.NewStdLogAt(zap.L(), zap.ErrorLevel)
	MQTT.CRITICAL, _ = zap.NewStdLogAt(zap.L(), zap.ErrorLevel)
	MQTT.WARN, _ = zap.NewStdLogAt(zap.L(), zap.WarnLevel)
}

func lateInitLogging(config *applicationConfig) {
	if *config.Debug {
		atomicLevel.SetLevel(zap.DebugLevel)
		MQTT.DEBUG, _ = zap.NewStdLogAt(zap.L(), zap.DebugLevel)
	}
}
