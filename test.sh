#!/usr/bin/env bash

# Load local settings if present. .env is gitignored and typically holds
# PISTON_API_KEY for the official API, or PISTON_BASE_URL for a self-hosted
# instance. Without it the live tests skip themselves rather than fail.
if [ -f .env ]; then
	set -a
	. ./.env
	set +a
fi

go clean -testcache && go test ./... -v -cover | sed ''/PASS/s//$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$(printf "\033[31mFAIL\033[0m")/'' | sed ''/RUN/s//$(printf "\033[33mRUN\033[0m")/'' | GREP_COLORS='mt=01;32' grep -E --color 'ok.*$|^.*ok.*$|$'
