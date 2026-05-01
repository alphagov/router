#!/usr/bin/env bash

set -euo pipefail

RAW_REPORT=$(mktemp)
COVERAGE_DIR=$(dirname "${BASH_SOURCE[0]}")

function emoji_for_percent {
  local PERCENT=$1

  INT_VALUE=$(printf "%.0f" "$PERCENT")

  if [ "$INT_VALUE" -ge 95 ]; then
    echo ":star:"
  elif [ "$INT_VALUE" -ge 90 ]; then
    echo ":white_tick:"
  elif [ "$INT_VALUE" -ge 80 ]; then
    echo ":red_circle:"
  else
    echo ":broken_heart:"
  fi
}

go tool covdata percent -i "${COVERAGE_DIR}/merged" > "$RAW_REPORT"

MARKDOWN_REPORT="${COVERAGE_DIR}/report/overall-coverage.md"
OVERALL_COVERAGE=$(awk '{ print substr($3, 1, length($3)-1); }' "${COVERAGE_DIR}/report/overall-coverage.txt")
OVERALL_EMOJI=$(emoji_for_percent "$OVERALL_COVERAGE")

cat > "$MARKDOWN_REPORT" <<EOF
## Coverage report

Overall coverage: ${OVERALL_COVERAGE}% ${OVERALL_EMOJI}

Package | Statements Covered | Happy?
-------|:------------------:|:------:
EOF

while read -r LINE; do
  PACKAGE=$(awk '{ print $1; '} <<<"$LINE")
  PERCENTAGE=$(awk '{ print substr($3, 1, length($3)-1); }' <<< "$LINE")
  EMOJI=$(emoji_for_percent "$PERCENTAGE")

  echo "${PACKAGE} | ${PERCENTAGE}% | $EMOJI" >> "$MARKDOWN_REPORT"
done < "$RAW_REPORT"
