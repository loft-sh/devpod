#!/bin/zsh

devpod ssh "load$1" --command="tr -dc A-Za-z0-9 </dev/urandom | head -c 1000000; echo" > /dev/null
