#!/bin/sh
set -eu

if [ "$#" -eq 0 ]; then
    set -- serve
fi

# Accept both ordinary image args (`serve`, `work:queues`) and the explicit
# spelling (`shrike serve`) people naturally use while debugging.
if [ "$1" = "shrike" ] || [ "$1" = "ek" ]; then
    shift
    if [ "$#" -eq 0 ]; then
        set -- serve
    fi
fi

# `shrike serve` owns and supervises Bun itself. Worker, cron, and one-shot
# commands use this exact same image but never start the renderer.
exec /usr/local/bin/shrike "$@"
