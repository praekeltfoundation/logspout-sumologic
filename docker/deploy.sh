#!/bin/sh
set -e
docker tag ${IMAGE} logspout-sumologic:${version}-logspout-${logspout_version}
docker tag ${IMAGE} logspout-sumologic:latest
docker push ${IMAGE}:${version}-logspout-${logspout_version}
docker push ${IMAGE}:latest
