package logger

import (
	"os"
)

type Conf struct {
	LogDir string
}

func LogConfig() *Conf {
	configs := new(Conf)
	logDir, set := os.LookupEnv("GOMCP_LOG_DIR")
	if !set {
		logDir, _ = os.Getwd()
	}
	configs.LogDir = logDir
	return configs
}

var logCfg = LogConfig()
