#!/bin/bash
# test-all.sh - Run all tests with race detection on Ubuntu
#
# Usage: ./scripts/test-all.sh [--no-race]

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

RACE="-race"
if [[ "$1" == "--no-race" ]]; then
  RACE=""
  shift
fi

# Run all tests in the codebase
${SCRIPT_DIR}/ubuntu-test.sh -v $RACE ./...