package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/function61/gokit/logger"
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/systemdinstaller"
	"github.com/function61/ruuvinator/pkg/hciframereceiver"
	"github.com/function61/ruuvinator/pkg/output/consoleoutput"
	"github.com/function61/ruuvinator/pkg/output/sqsoutput"
	"github.com/function61/ruuvinator/pkg/ruuviframeparser"
	"github.com/function61/ruuvinator/pkg/ruuvinatortypes"
	"github.com/spf13/cobra"
	"os"
)

func client() error {
	log := logger.New("main loop")
	log.Info("starting")
	defer log.Info("stopped")

	conf, err := readConfig()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var output ruuvinatortypes.Output

	switch conf.Output {
	case "sqsoutput":
		output = sqsoutput.New(ctx, *conf.SqsOutputConfig)
	case "console":
		output = consoleoutput.New()
	default:
		panic(errors.New("unknown output: " + conf.Output))
	}

	observationsCh := output.GetObservationsChan()

	go func() {
		log.Info(fmt.Sprintf("got %s; stopping", ossignal.WaitForInterruptOrTerminate()))

		// stops all subprocesses
		cancel()
	}()

	sensorResolver := NewWhitelistResolver(conf.SensorWhitelist)

	hciframereceiver.Run(ctx, func(frame hciframereceiver.Frame) {
		// don't bother logging errors, as there is a lot of non-Ruuvi traffic over the air
		observation, _ := ruuviframeparser.Parse(frame)
		if observation == nil {
			return
		}

		resolvedObservation, ok := sensorResolver.Resolve(*observation)
		if !ok {
			log.Info(fmt.Sprintf("unknown Ruuvi traffic from %s", observation.SensorAddr))
			return
		}

		if observation != nil {
			observationsCh <- *resolvedObservation
		}
	})

	return nil
}

func clientEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "Listen for Ruuvi frames over Bluetooth and send them to configured output",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := client(); err != nil {
				panic(err)
			}
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "write-systemd-unit-file",
		Short: "Install unit file to start Ruubinator Bluetooth listener on startup",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			systemdHints, err := systemdinstaller.InstallSystemdServiceFile("ruuvinator-client", []string{"client"}, "Ruuvinator client")
			if err != nil {
				panic(err)
			}

			fmt.Println(systemdHints)
		},
	})

	return cmd
}

func readConfig() (*ruuvinatortypes.Config, error) {
	file, err := os.Open("config.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	jsonDecoder := json.NewDecoder(file)
	jsonDecoder.DisallowUnknownFields()

	conf := &ruuvinatortypes.Config{}
	if err := jsonDecoder.Decode(conf); err != nil {
		return nil, err
	}

	return conf, nil
}
