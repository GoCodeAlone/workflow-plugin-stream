#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
GOWORK=off go test ./session -run TestServiceSessionSuperviseRenewMutateDrainAndUrgentKill -count=1 -v
