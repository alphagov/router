#!/usr/bin/env bash

set -euo pipefail

RAW_REPORT=$(mktemp)
COVERAGE_DIR=$(dirname "${BASH_SOURCE[0]}")
UNKNOWN_COVERAGE="Unknown"

function positive_or_negative_change {
  # Args:
  #   $1: Percentage points change
  #
  # Echos:
  #   1 if coverage change is positive
  #   0 if no change
  #   -1 if change is negative
  local PERCENTAGE_POINTS_CHANGE=$1

  if [ "$PERCENTAGE_POINTS_CHANGE" == "$UNKNOWN_COVERAGE" ]; then
    echo "$UNKNOWN_COVERAGE"
    return 0
  fi

  bc <<<"change=${PERCENTAGE_POINTS_CHANGE}; if(change > 0) 1 else if(change < 0) -1 else 0;"
}

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

function emoji_for_change {
  # Args:
  #   $1: Must be 1 (for positive change), 0 (for no change), -1 (for negative change), or Unknown
  local CHANGE_DIRECTION=$1

  case "$CHANGE_DIRECTION" in
    "$UNKNOWN_COVERAGE")
      echo ":grey_question:"
      ;;
    "1")
      echo ":point_up:"
      ;;
    "0")
      echo ":point_left:"
      ;;
    "-1")
      echo ":point_down:"
      ;;
    *)
      echo ":grey_question:"
      ;;
  esac
}

function overall_coverage_percent_on_main {
  if [ ! -f "${COVERAGE_DIR}/report-main/overall-coverage.txt" ]; then
    echo "$UNKNOWN_COVERAGE"
    return 0
  fi

  awk '{ print substr($3, 1, length($3)-1); }' "${COVERAGE_DIR}/report/overall-coverage.txt"
}

function coverage_change {
  # Args:
  #   $1: Coverage % on main
  #   $2: Coverage % now
  local COVERAGE_ON_MAIN=$1
  local COVERAGE_NOW=$1

  if [ "$COVERAGE_ON_MAIN" == "$UNKNOWN_COVERAGE" ]; then
    echo "$UNKNOWN_COVERAGE"
    return 0
  fi

  bc <<<"$COVERAGE_NOW - $COVERAGE_ON_MAIN"
}

function change_percentage_text {
  local PERCENTAGE_POINTS_CHANGE=$1
  local CHANGE_DIRECTION
  local EMOJI

  CHANGE_DIRECTION=$(positive_or_negative_change "$PERCENTAGE_POINTS_CHANGE")

  EMOJI=$(emoji_for_change "$CHANGE_DIRECTION")

  case "$CHANGE_DIRECTION" in
    "$UNKNOWN_COVERAGE")
      echo "Unknown ${EMOJI}"
      ;;
    "1")
      echo "+${PERCENTAGE_POINTS_CHANGE}% ${EMOJI}"
      ;;
    "0")
      echo "+/- 0% ${EMOJI}"
      ;;
    "-1")
      echo "${PERCENTAGE_POINTS_CHANGE}% ${EMOJI}"
      ;;
    *)
      echo "Unknown ${EMOJI}"
      ;;
  esac
}

go tool covdata percent -i "${COVERAGE_DIR}/merged" > "$RAW_REPORT"

MARKDOWN_REPORT="${COVERAGE_DIR}/report/overall-coverage.md"

OVERALL_COVERAGE=$(awk '{ print substr($3, 1, length($3)-1); }' "${COVERAGE_DIR}/report/overall-coverage.txt")
OVERALL_COVERAGE_ON_MAIN=$(overall_coverage_percent_on_main)
OVERALL_COVERAGE_CHANGE_VS_MAIN=$(coverage_change "$OVERALL_COVERAGE_ON_MAIN" "$OVERALL_COVERAGE")

OVERALL_EMOJI=$(emoji_for_percent "$OVERALL_COVERAGE")

WARNING_TEXT=""
if [ "${OVERALL_COVERAGE_ON_MAIN}" == "$UNKNOWN_COVERAGE" ]; then
  WARNING_TEXT="
***Warning***: Coverage report for main not found, cannot compare with previous coverage
"
fi

cat > "$MARKDOWN_REPORT" <<EOF
## Coverage report
$WARNING_TEXT
Overall coverage: ${OVERALL_COVERAGE}% ${OVERALL_EMOJI} ($(change_percentage_text "$OVERALL_COVERAGE_CHANGE_VS_MAIN"))

Package | Statements Covered | Happy?
-------|:------------------:|:------:
EOF

while read -r LINE; do
  PACKAGE=$(awk '{ print $1; }' <<<"$LINE")
  PERCENTAGE=$(awk '{ print substr($3, 1, length($3)-1); }' <<< "$LINE")
  EMOJI=$(emoji_for_percent "$PERCENTAGE")

  echo "${PACKAGE} | ${PERCENTAGE}% | $EMOJI" >> "$MARKDOWN_REPORT"
done < "$RAW_REPORT"
