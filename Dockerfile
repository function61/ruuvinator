FROM alpine:3.8

# needed for AWS SQS
RUN apk add ca-certificates

CMD ["ruuvinator", "metricsserver"]

COPY rel/ruuvinator_linux-amd64 /usr/local/bin/ruuvinator
