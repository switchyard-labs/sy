#!/usr/bin/env bash
# Opt-in live integration test (client-only; never during default unit tests).
#   SY_INTEGRATION=1 scripts/integration.sh
# Uses only sy-dogfood-* disposable state.
set -uo pipefail
SY_HOME="$(mktemp -d)"
export SY_CONFIG_DIR="$SY_HOME"
HOST="${SY_HOST:-http://45.79.189.46}"
USER="${SY_USER:-alice}"
TS=$(date +%s)
BIN=${SY_BIN:-"go run ./cmd/sy"}
pass=0; fail=0
ok(){ echo "PASS $1"; pass=$((pass+1)); }
bad(){ echo "FAIL $1 — $2"; fail=$((fail+1)); }

$BIN auth login --host "$HOST" --username "$USER" --password-stdin >/dev/null 2>&1 <<< "$USER
$SY_PASSWORD" || { echo "need SY_PASSWORD for $USER"; exit 1; }
$BIN auth status >/dev/null 2>&1 && ok "auth login+status" || bad "auth" "login failed"

n=$($BIN repo list --json | jq '.items | length')
[ "${n:-0}" -ge 1 ] && ok "repo list ($n)" || bad "repo list" "empty"

WID=$($BIN work create --title "sy-dogfood-$TS" --kind task --json | jq -r .id)
[ -n "$WID" ] && ok "work create ($WID)" || bad "work create" "no id"
$BIN work view "$WID" --json >/dev/null 2>&1 && ok "work view" || bad "work view" "404"
$BIN work comment "$WID" --body "integration harness" >/dev/null 2>&1 && ok "work comment" || bad "work comment" "api"
$BIN work close "$WID" >/dev/null 2>&1 && ok "work close" || bad "work close" "api"

$BIN attention --json >/dev/null 2>&1 && ok "attention" || bad "attention" "api"
$BIN queue list --json >/dev/null 2>&1 && ok "queue list" || bad "queue list" "api"
$BIN workflow list --json >/dev/null 2>&1 && ok "workflow list" || bad "workflow list" "api"
$BIN agent roles --json >/dev/null 2>&1 && ok "agent roles" || bad "agent roles" "api"
$BIN org list --json >/dev/null 2>&1 && ok "org list" || bad "org list" "api"

echo "integration: $pass pass, $fail fail"
[ "$fail" -eq 0 ]
