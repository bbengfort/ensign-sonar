package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	sonar "github.com/bbengfort/ensign-sonar"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func main() {
	// If a dotenv file exists load it for configuration
	godotenv.Load()

	// Create a multi-command CLI application
	app := cli.NewApp()
	app.Name = "ensign-debug"
	app.Version = sonar.Version()
	app.Before = setupLogger
	app.Usage = "sends and receives ping events to test ensign connectivity"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "verbosity",
			Aliases: []string{"L"},
			Usage:   "set the zerolog level",
			Value:   "info",
			EnvVars: []string{"ENSIGN_LOG_LEVEL"},
		},
		&cli.BoolFlag{
			Name:    "console",
			Aliases: []string{"C"},
			Usage:   "human readable console log instead of json",
			Value:   false,
			EnvVars: []string{"ENSIGN_CONSOLE_LOG"},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("could not execute cli app")
	}
}

func setupLogger(c *cli.Context) (err error) {
	switch strings.ToLower(c.String("verbosity")) {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn", "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		return cli.Exit(fmt.Errorf("unknown log level %q", c.String("verbosity")), 1)
	}

	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.DurationFieldInteger = false
	zerolog.DurationFieldUnit = time.Millisecond

	if c.Bool("console") {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	return nil
}
