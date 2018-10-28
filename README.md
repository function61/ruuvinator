[![Build Status](https://img.shields.io/travis/function61/ruuvinator.svg?style=for-the-badge)](https://travis-ci.org/function61/ruuvinator)
[![Download](https://img.shields.io/bintray/v/function61/ruuvinator/main.svg?style=for-the-badge&label=Download)](https://bintray.com/function61/ruuvinator/main/_latestVersion#files)
[![Download](https://img.shields.io/docker/pulls/fn61/ruuvinator.svg?style=for-the-badge)](https://hub.docker.com/r/fn61/ruuvinator/)

[Ruuvitag](https://shop.ruuvi.com/product/ruuvitag/) Bluetooth listener ("client") &
[Prometheus](https://prometheus.io/) metrics server ("server").

Client - server model communicates via [AWS SQS](https://aws.amazon.com/sqs/), so the
[Raspberry Pi](https://www.raspberrypi.org/) I use to listen to Ruuvi traffic doesn't have
to have anything extra.

The client has pluggable outputs:

- Print to console (doesn't need the server component at all)
- AWS SQS

The client tries its best to send observations in one-second batches so one client shouldn't
do much more than 86 400 requests to SQS a day even if you have more trackers than three.


How to build & develop
----------------------

[How to build & develop](https://github.com/function61/turbobob/blob/master/docs/external-how-to-build-and-dev.md)
(with Turbo Bob, our build tool). It's easy and simple!


Usage, client
-------------

Download suitable binary for your architecture from Bintray download link from the top of
this README.

Configure `config.json`. Example with SQS:

```
{
	"sensor_whitelist": {
		"aa:bb:cc:dd:ee:ff": "Bedroom",
		"ff:ee:dd:cc:bb:aa": "Outside"
	},
	"output": "sqsoutput",
	"sqsoutput_config": {
		"queue_url": "https://sqs.us-east-1.amazonaws.com/123456789/Ruuvinator",
		"aws_access_key_id": "AKIA...",
		"aws_access_key_secret": "E+mEut..."
	}
}
```

Don't worry if you don't know your sensors' Bluetooth addresses. For non-whitelisted Ruuvis,
you can find log lines like these:

```
observation from unknown Ruuvi fb:72:36:09:90:15
```

Example config with just printing to console:

```
{
	"sensor_whitelist": {
		"aa:bb:cc:dd:ee:ff": "Bedroom"
	},
	"output": "console"
}
```

Now try running it (you might need to run it with sudo):

```
$ ./ruuvinator client
```

To make it start on system startup - you'll get nice help tips as well:

```
$ ./ruuvinator client write-systemd-unit-file
Wrote unit file to /etc/systemd/system/ruuvinator-client.service
Run to enable on boot & to start now:
	$ systemctl enable ruuvinator-client
	$ systemctl start ruuvinator-client
```

Troubleshooting: if Bluetooth gives you grief,
[have you tried turning it off and on again](https://youtu.be/nn2FB1P_Mn8?t=10)?

```
$ hciconfig hci0 down && hciconfig hci0 up
```


Usage, server
-------------

You can find the Docker image from the Docker link from the top of this README.

The server is designed to run as a Docker container. Define these ENV variables:

- `QUEUE_URL`
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`

Prometheus metrics will be available at `http://ip/metrics`
