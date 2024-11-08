#!/bin/zsh

devpod ssh "load$1" --command="cat /var/log/bootstrap.log" > /dev/null