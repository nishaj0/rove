package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"cloud-saver/backend/internal/app"
)

func main() {
	var configPath string
	var noTray bool
	var debug bool
	flag.StringVar(&configPath, "config", "", "path to settings.json")
	flag.BoolVar(&noTray, "no-tray", false, "run without system tray")
	flag.BoolVar(&debug, "debug", false, "enable verbose debug logs")
	flag.Parse()

	if !debug {
		debug = os.Getenv("CLOUD_SAVER_DEBUG") == "1"
	}

	application, err := app.New(app.Options{ConfigPath: configPath, NoTray: noTray, Debug: debug})
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	// tie app lifetime to ctrl+c and termination signals.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := application.Run(ctx, noTray); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
