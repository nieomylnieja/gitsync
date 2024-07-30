package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

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
		defaultConfigPath := getDefaultConfigPath()
		if _, err := os.Stat(defaultConfigPath); err != nil {
			_, _ = fmt.Fprintf(flag.CommandLine.Output(),
				"error: '-c' was not provided and there was no default config file located at: %s\n",
				defaultConfigPath)
			flag.Usage()
			os.Exit(1)
		} else {
			configPath = &defaultConfigPath
		}
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

func getDefaultConfigPath() string {
	var path string
	if xdgConfigHome, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		path = filepath.Join(xdgConfigHome, "gitsync", "config.json")
	} else {
		path = os.ExpandEnv(filepath.Join("$HOME", ".config", "gitsync", "config.json"))
	}
	return path
}
