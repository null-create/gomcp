package server

import (
	"time"
)

type Conf struct {
	TimeoutRead  time.Duration
	TimeoutWrite time.Duration
	TimeoutIdle  time.Duration
}

func ServerConfigs() *Conf {
	return &Conf{
		TimeoutRead:  time.Second * 30,
		TimeoutWrite: time.Second * 30,
		TimeoutIdle:  time.Second * 30,
	}
}
