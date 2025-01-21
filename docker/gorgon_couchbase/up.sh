#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

cd $(dirname "$0")
pwd

make

docker-compose -f docker-compose.yml up $*
