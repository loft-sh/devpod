#!/bin/sh

# Write environment variable to file when entrypoint is executed
echo "RESULT_ENV=${TEST_FEATURE_ENV}" > /tmp/feature-entrypoint-env.txt

# Continue with original entrypoint
exec $@
