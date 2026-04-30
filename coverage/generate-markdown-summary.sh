#!/usr/bin/env bash

set -euo pipefail

RAW_REPORT=$(mktemp)
COVERAGE_DIR=$(dirname "${BASH_SOURCE[0]}")

go tool covdata percent -i "${COVERAGE_DIR}/merged" > "$RAW_REPORT"

MARKDOWN_REPORT="${COVERAGE_DIR}/report/overall-coverage.md"
cat > "$MARKDOWN_REPORT" <<EOF
## Coverage report

Module | Statements Covered | Happy?
-------|:------------------:|:------:
EOF

function emoji_for_percent {
  local PERCENT=$1

  INT_VALUE=$(printf "%.0f" "$PERCENT")

  if [ "$INT_VALUE" -ge 90 ]; then
    echo ":heart_eyes:"
  elif [ "$INT_VALUE" -ge 80 ]; then
    echo ":smile:"
  elif [ "$INT_VALUE" -ge 70 ]; then
    echo ":sweat:"
  else
    echo ":fearful:"
  fi
}

while read -r LINE; do
  PACKAGE=$(awk '{ print $1; '} <<<"$LINE")
  PERCENTAGE=$(awk '{ print substr($3, 1, length($3)-1); }' <<< "$LINE")
  EMOJI=$(emoji_for_percent "$PERCENTAGE")

  echo "${PACKAGE} | ${PERCENTAGE}% | $EMOJI" >> "$MARKDOWN_REPORT"
done < "$RAW_REPORT"

OVERALL_COVERAGE=$(awk '{ print substr($3, 1, length($3)-1); }' <"${COVERAGE_DIR}/report/overall-coverage.txt")
OVERALL_EMOJI=$(emoji_for_percent "$OVERALL_COVERAGE")

echo -e "\n\nOverall coverage: ${OVERALL_COVERAGE}% ${OVERALL_EMOJI}\n" >> "$MARKDOWN_REPORT"
