package main

import (
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

func init() {
	//initialise our global logger

	logger, _ := zap.NewDevelopment(
		zap.AddStacktrace(zap.FatalLevel), //disable stacktrace for level lower than fatal
	)
	zap.ReplaceGlobals(logger)

	MQTT.ERROR, _ = zap.NewStdLogAt(zap.L(), zap.ErrorLevel)
	MQTT.CRITICAL, _ = zap.NewStdLogAt(zap.L(), zap.ErrorLevel)
	MQTT.WARN, _ = zap.NewStdLogAt(zap.L(), zap.WarnLevel)
	MQTT.DEBUG, _ = zap.NewStdLogAt(zap.L(), zap.DebugLevel)
}
