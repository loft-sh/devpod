#!/bin/sh

script_dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

install -D -m 755 "$script_dir/entrypoint.sh" /usr/local/bin/test-feature-entrypoint.sh

echo "Test feature installed successfully"
