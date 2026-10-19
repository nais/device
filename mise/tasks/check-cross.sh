#!/usr/bin/env bash
#MISE description="Run all checks for linux, darwin and windows (CI only checks linux)"
set -o errexit -o nounset -o pipefail

CHECK_GOOS="linux darwin windows" exec mise run check
