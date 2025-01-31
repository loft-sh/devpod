#!/bin/bash
set -e

docker build -f linux.cover.dockerfile -t linux.cover.djherbis.times .
docker create --name linux.cover.djherbis.times linux.cover.djherbis.times
docker cp linux.cover.djherbis.times:/go/src/github.com/djherbis/times/profile.cov .
docker rm -v linux.cover.djherbis.times