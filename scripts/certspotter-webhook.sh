#!/bin/bash
# Certspotter webhook script - forwards certificate data to Canary
set -euo pipefail

CANARY_ENDPOINT="${CANARY_ENDPOINT:-http://localhost:8080/hook}"

curl -s -X POST \
    -H "Content-Type: application/json" \
    -d @- \
    "$CANARY_ENDPOINT"
