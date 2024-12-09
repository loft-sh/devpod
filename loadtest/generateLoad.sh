#!/bin/zsh

devpod ssh "loadtest$1" --command="tr -dc A-Za-z0-9 </dev/urandom | head -c 100000000; echo" > /dev/null
