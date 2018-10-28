package main

import (
	"encoding/json"
	"fmt"
	"github.com/function61/gokit/envvar"
	"github.com/function61/gokit/logger"
	"github.com/function61/ruuvinator/pkg/ruuvinatortypes"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"net/http"
	"time"
)

// TODO: backoff

func metricsServer(conf ruuvinatortypes.SqsOutputConfig) error {
	log := logger.New("metrics-server")

	temperature, humidity, pressure, battery, accelerationSum := initializeMetrics()

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		log.Error(http.ListenAndServe(":80", nil).Error())
	}()

	sqsFacade := NewSQS(conf.QueueUrl, conf.AwsAccessKeyId, conf.AwsAccessKeySecret)

	for {
		received, err := sqsFacade.Receive()
		if err != nil {
			log.Error(err.Error())
			time.Sleep(1 * time.Second) // prevent hot loop
			continue
		}

		for _, item := range received.Messages {
			msg := &ruuvinatortypes.ResolvedSensorObservation{}
			if err := json.Unmarshal([]byte(*item.Body), msg); err != nil {
				// TODO: do not ack this, but let it error so long that it goes in the DLQ
				log.Error(fmt.Sprintf("error processing %s", *item.Body))
				continue
			}

			sensorLabels := prometheus.Labels{
				"sensor": msg.Observation.SensorAddr,
				"name":   msg.SensorName,
			}

			measurements := msg.Observation.Measurements // shorthand

			temperature.With(sensorLabels).Set(measurements.Temperature)
			humidity.With(sensorLabels).Set(measurements.Humidity)
			battery.With(sensorLabels).Set(measurements.Battery)
			pressure.With(sensorLabels).Set(float64(measurements.Pressure))
			accelerationSum.With(sensorLabels).Set(float64(measurements.Acceleration.X +
				measurements.Acceleration.Y +
				measurements.Acceleration.Z))
		}

		if err := sqsFacade.AckReceived(received); err != nil {
			// TODO: retry?
			log.Error(err.Error())
		}
	}
}

func metricsServerEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "metricsserver",
		Short: "Serves metrics downloaded from SQS messages",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			conf, err := getConfigFromEnv()
			if err != nil {
				panic(err)
			}

			if err := metricsServer(*conf); err != nil {
				panic(err)
			}
		},
	}
}

func getConfigFromEnv() (*ruuvinatortypes.SqsOutputConfig, error) {
	queueUrl, err := envvar.Get("QUEUE_URL")
	if err != nil {
		return nil, err
	}

	accessKeyId, err := envvar.Get("AWS_ACCESS_KEY_ID")
	if err != nil {
		return nil, err
	}

	accessKeySecret, err := envvar.Get("AWS_SECRET_ACCESS_KEY")
	if err != nil {
		return nil, err
	}

	return &ruuvinatortypes.SqsOutputConfig{
		QueueUrl:           queueUrl,
		AwsAccessKeyId:     accessKeyId,
		AwsAccessKeySecret: accessKeySecret,
	}, nil
}

func initializeMetrics() (*prometheus.GaugeVec, *prometheus.GaugeVec, *prometheus.GaugeVec, *prometheus.GaugeVec, *prometheus.GaugeVec) {
	labels := []string{"sensor", "name"}

	temperature := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ruuvi_temperature",
			Help: "Ruuvi: temperature",
		},
		labels)
	prometheus.MustRegister(temperature)

	humidity := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ruuvi_humidity",
			Help: "Ruuvi: humidity",
		},
		labels)
	prometheus.MustRegister(humidity)

	pressure := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ruuvi_pressure",
			Help: "Ruuvi: pressure",
		},
		labels)
	prometheus.MustRegister(pressure)

	battery := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ruuvi_battery",
			Help: "Ruuvi: battery",
		},
		labels)
	prometheus.MustRegister(battery)

	accelerationSum := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ruuvi_acceleration_sum",
			Help: "Ruuvi: acceleration x + y + z",
		},
		labels)
	prometheus.MustRegister(accelerationSum)

	return temperature, humidity, pressure, battery, accelerationSum
}
