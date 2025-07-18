#!/bin/sh
set -e

ARGS=""

[ -n "$BAUD" ] && ARGS="$ARGS -baud $BAUD"
[ -n "$CITY" ] && ARGS="$ARGS -city $CITY"
[ "$DASHBOARD" = "true" ] && ARGS="$ARGS -dashboard"
[ -n "$DB" ] && ARGS="$ARGS -db $DB"
[ -n "$EXPORT_CSV" ] && ARGS="$ARGS -export-csv $EXPORT_CSV"
[ -n "$LOG_FILE" ] && ARGS="$ARGS -log-file $LOG_FILE"
[ -n "$PORT" ] && ARGS="$ARGS -port $PORT"
[ "$WEATHER" = "true" ] && ARGS="$ARGS -weather"

exec ./skogsnet_v2 $ARGS
