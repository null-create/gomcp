package logger

import (
	"log"
	"os"

	"github.com/joeshaw/envdecode"
)

type Conf struct {
	LogDir string `env:"GOMCP_LOG_DIR,required"`
}

func LogConfig() *Conf {
	configs := new(Conf)

	logDir := os.Getenv("GOMCP_LOG_DIR")
	if logDir == "" {
		if err := envdecode.StrictDecode(configs); err != nil {
			log.Fatalf("failed to decode log config .env file: %s", err)
		}
	} else {
		configs.LogDir = logDir
	}

	return configs
}

var logCfg = LogConfig()
