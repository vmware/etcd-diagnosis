#!/usr/bin/env bash

set -euo pipefail

####   Convenient IO methods #####

export COLOR_RED='\033[0;31m'
export COLOR_ORANGE='\033[0;33m'
export COLOR_GREEN='\033[0;32m'
export COLOR_LIGHTCYAN='\033[0;36m'
export COLOR_BLUE='\033[0;94m'
export COLOR_BOLD='\033[1m'
export COLOR_MAGENTA='\033[95m'
export COLOR_NONE='\033[0m' # No Color


function log_error {
  >&2 echo -n -e "${COLOR_BOLD}${COLOR_RED}"
  >&2 echo "$@"
  >&2 echo -n -e "${COLOR_NONE}"
}

function log_warning {
  >&2 echo -n -e "${COLOR_ORANGE}"
  >&2 echo "$@"
  >&2 echo -n -e "${COLOR_NONE}"
}

function log_callout {
  >&2 echo -n -e "${COLOR_LIGHTCYAN}"
  >&2 echo "$@"
  >&2 echo -n -e "${COLOR_NONE}"
}

function log_cmd {
  >&2 echo -n -e "${COLOR_BLUE}"
  >&2 echo "$@"
  >&2 echo -n -e "${COLOR_NONE}"
}

function log_success {
  >&2 echo -n -e "${COLOR_GREEN}"
  >&2 echo "$@"
  >&2 echo -n -e "${COLOR_NONE}"
}

function log_info {
  >&2 echo -n -e "${COLOR_NONE}"
  >&2 echo "$@"
  >&2 echo -n -e "${COLOR_NONE}"
}
