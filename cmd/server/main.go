package main

import (
	"github.com/johnfarrell/runeplan/config"
	"github.com/johnfarrell/runeplan/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log, err := loggerd.New(cfg.App)
	if err != nil {
		panic(err)
	}
	defer log.Sync() //nolint:errcheck

	log.Info("runeplan starting")
}
