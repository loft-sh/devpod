#!/usr/bin/env bash

set -e

export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get install -y hello
