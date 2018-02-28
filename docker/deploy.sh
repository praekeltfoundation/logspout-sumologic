#!/bin/sh
set -e
docker tag ${IMAGE} ${IMAGE}:${version}-logspout-${logspout_version}
docker tag ${IMAGE} ${IMAGE}:latest
docker push ${IMAGE}:${version}-logspout-${logspout_version}
docker push ${IMAGE}:latest
