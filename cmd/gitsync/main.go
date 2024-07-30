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
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Usage: gitsync [options] [command]")
		flag.PrintDefaults()
	}
	configPath := flag.String("c", "", "path to the configuration file")
	flag.Parse()
	if flag.NArg() != 1 {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(),
			"error: invalid number of arguments, provide either 'sync' or 'diff' command")
		flag.Usage()
		os.Exit(1)
	}
	var command gitsync.Command
	switch flag.Arg(0) {
	case "sync":
		command = gitsync.CommandSync
	case "diff":
		command = gitsync.CommandDiff
	default:
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "error: invalid command, provide either 'sync' or 'diff'")
		flag.Usage()
		os.Exit(1)
	}
	if configPath == nil || *configPath == "" {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "error: '-c' flag is required but was not provided")
		flag.Usage()
		os.Exit(1)
	}
	conf, err := config.ReadConfig(*configPath)
	if err != nil {
		return err
	}
	if err = gitsync.Run(conf, command); err != nil {
		return err
	}
	if err = conf.Save(); err != nil {
		return err
	}
	return nil
}
