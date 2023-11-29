#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

go run main.go docs
