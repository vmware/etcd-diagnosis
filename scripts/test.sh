#!/usr/bin/env bash

set -e
set -o pipefail
set -o nounset

source ./scripts/test_utils.sh

# generic_checker [cmd...]
# executes given command in the current module, and clearly fails if it
# failed or returned output.
function generic_checker {
  local cmd=("$@")
  if ! output=$("${cmd[@]}"); then
    echo "${output}"
    log_error -e "FAIL: '${cmd[*]}' checking failed (!=0 return code)"
    return 255
  fi
  if [ -n "${output}" ]; then
    echo "${output}"
    log_error -e "FAIL: '${cmd[*]}' checking failed (printed output)"
    return 255
  fi
}

function mod_tidy_pass {
  generic_checker go mod tidy -diff
}

########### MAIN ###############################################################

function run_pass {
  local pass="${1}"
  shift 1
  log_callout -e "\\n'${pass}' started at $(date)"
  if "${pass}_pass" "$@" ; then
    log_success "'${pass}' PASSED and completed at $(date)"
    return 0
  else
    log_error "FAIL: '${pass}' FAILED at $(date)"
    if [ "$KEEP_GOING_SUITE" = true ]; then
      return 2
    else
      exit 255
    fi
  fi
}

log_callout "Starting at: $(date)"
fail_flag=false
for pass in $PASSES; do
  if run_pass "${pass}" "$@"; then
    continue
  else
    fail_flag=true
  fi
done
if [ "$fail_flag" = true ]; then
  log_error "There was FAILURE in the test suites ran. Look above log detail"
  exit 255
fi

log_success "SUCCESS"
