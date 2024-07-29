package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nieomylnieja/gitsync/internal/config"
	"github.com/nieomylnieja/gitsync/internal/gitsync"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Usage = func() {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Usage: gitsync [options]")
		flag.PrintDefaults()
	}
	flag.String("c", "config.json", "path to the configuration file")
	flag.Parse()
	conf, err := config.ReadConfig("config.json")
	if err != nil {
		return err
	}
	if err = gitsync.Run(conf); err != nil {
		return err
	}
	if err = conf.Save(); err != nil {
		return err
	}
	return nil
}
