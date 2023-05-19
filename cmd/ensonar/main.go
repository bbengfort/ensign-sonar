package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	sonar "github.com/bbengfort/ensign-sonar"
	"github.com/joho/godotenv"
	"github.com/rotationalio/go-ensign"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var client *ensign.Client

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
			Name:    "topic",
			Aliases: []string{"t"},
			Usage:   "specify the sonar topic to use",
			Value:   "sonar.ping",
			EnvVars: []string{"ENSIGN_SONAR_TOPIC"},
		},
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
	app.Commands = []*cli.Command{
		{
			Name:   "sonar",
			Usage:  "generate sonar pings and send to the specified topic",
			Before: connect,
			After:  disconnect,
			Action: runSonar,
			Flags: []cli.Flag{
				&cli.Float64Flag{
					Name:    "rate",
					Aliases: []string{"r"},
					Usage:   "events to publish per second (-1 for as fast as possible)",
					Value:   30,
				},
			},
		},
		{
			Name:   "listen",
			Usage:  "subscribe to the stream and listen for sonar pings",
			Before: connect,
			After:  disconnect,
			Action: listen,
			Flags:  []cli.Flag{},
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

func connect(c *cli.Context) (err error) {
	if client, err = ensign.New(); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}

func disconnect(c *cli.Context) (err error) {
	if err = client.Close(); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}

func runSonar(c *cli.Context) (err error) {
	pings := sonar.New()
	topic := c.String("topic")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	count := uint64(0)

	var exists bool
	if exists, err = client.TopicExists(context.Background(), topic); err != nil {
		return cli.Exit(err, 1)
	}

	var topicID string
	if !exists {
		if topicID, err = client.CreateTopic(context.Background(), topic); err != nil {
			return cli.Exit(err, 1)
		}
	} else {
		if topicID, err = client.TopicID(context.Background(), topic); err != nil {
			return cli.Exit(err, 1)
		}
	}

	if hz := c.Float64("rate"); hz > 0 {
		interval := time.Duration(float64(time.Second) / hz)
		log.Info().Str("topic", topic).Float64("hz", hz).Dur("interval", interval).Msg("starting rate limited publisher")

		ticker := time.NewTicker(interval)
		for {
			select {
			case <-quit:
				fmt.Println("")
				return nil
			case <-ticker.C:
				count++
				if count%64 == 0 {
					fmt.Print("\033[2K\r")
				}

				ping := pings.Next().Event()
				if err = client.Publish(topicID, ping); err != nil {
					fmt.Print("x")
					log.Error().Err(err).Msg("could not publish ping")
					continue
				}
				
				if ping.Acked() {
					fmt.Print(".")
				}
			}
		}
	} else {
		log.Info().Str("topic", topic).Msg("starting max rate publisher")
		for {
			select {
			case <-quit:
				fmt.Println("")
				return nil
			default:
			}

			count++
			if count%64 == 0 {
				fmt.Print("\033[2K\r")
			}

			ping := pings.Next().Event()
			if err = client.Publish(topicID, ping); err != nil {
				fmt.Print("x")
				log.Error().Err(err).Msg("could not publish ping")
				continue
			}

			if ping.Acked() {
				fmt.Print(".")
			}
		}
	}
}

func listen(c *cli.Context) (err error) {
	var sub *ensign.Subscription
	if sub, err = client.Subscribe(); err != nil {
		return cli.Exit(err, 1)
	}
	defer sub.Close()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	for {
		select {
		case event := <-sub.C:
			var ping *sonar.Ping
			if err = ping.Unmarshal(event.Data); err != nil {
				log.Error().Err(err).Str("type", event.Type.String()).Str("mimetype", event.Mimetype.String()).Msg("could not unmarshal ping")
				event.Nack()
				continue
			}
			fmt.Println(ping.String())
			event.Ack()
		case <-quit:
			return nil
		}
	}
}
