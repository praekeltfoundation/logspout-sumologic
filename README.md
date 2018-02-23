# logspout-sumologic

[![Build Status](https://travis-ci.org/praekeltfoundation/logspout-sumologic.svg?branch=master)](https://travis-ci.org/praekeltfoundation/logspout-sumologic)

A [Logspout](https://github.com/gliderlabs/logspout) adapter for sending Docker container logs to a [Sumo Logic HTTP source ](https://help.sumologic.com/Send-Data/Sources/02Sources-for-Hosted-Collectors/HTTP-Source/Upload-Data-to-an-HTTP-Source).

## Usage Example:

```
sudo docker run --name logspout \
 -v /var/run/docker.sock:/var/run/docker.sock \
 -e SUMOLOGIC_SOURCE_NAME="{{index .Container.Config.Labels \"MESOS_TASK_ID\"}}" \
 -e SUMOLOGIC_ENDPOINT=https://collectors.de.sumologic.com/receiver/v1/http/ \
 praekeltfoundation/logspout-sumologic sumologic://

```

## Configuration:

Right now this adapter ignores any route address or route options provided by the command-line arguments and instead relies on environment variables for configuration.

```
SUMOLOGIC_ENDPOINT - e.g: https://collectors.de.sumologic.com/receiver/v1/http/Zm9vCg==
SUMOLOGIC_SOURCE_NAME - (Per container templateable) e.g
 {{.Container.Name}} (Default), {{index .Container.Config.Labels \"MESOS_TASK_ID\"}}
 or just a plain string.
SUMOLOGIC_SOURCE_CATEGORY - e.g qa/containers/myorg/frontend, also per container templateable.
SUMOLOGIC_SOURCE_HOST - {{.Container.Config.Hostname}} (default)
SUMOLOGIC_RETRIES - How many times to retry sending a log to the Sumo Logic http endpoint. defaults to 2
SUMOLOGIC_BACKOFF # TODO, defaults to 10
SUMOLOGIC_TIMEOUT_MS # TODO, defaults to 10000
```

## Building:
```
docker build -t logspout-sumologic .
```

## Testing:
```
go test -p 1 -v -coverprofile foo.out github.com/praekeltfoundation/logspout-sumologic
```
