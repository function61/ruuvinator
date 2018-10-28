#!/bin/bash -eu

source /build-common.sh

BINARY_NAME="ruuvinator"
COMPILE_IN_DIRECTORY="cmd/ruuvinator"
BINTRAY_PROJECT="function61/ruuvinator"

# aws has non-gofmt code..
GOFMT_TARGETS="cmd/ pkg/"

standardBuildProcess
