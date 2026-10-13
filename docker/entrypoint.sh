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

# All HTTP surfaces, including page rendering, run inside Shrike.
exec /usr/local/bin/shrike "$@"
