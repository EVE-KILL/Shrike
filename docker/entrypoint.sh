#!/bin/sh
set -u

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

# Find the Cobra command while allowing persistent flags to precede it, as in
# `--config /run/secrets/evekill.env serve`.
command_name=
skip_next=0
for arg in "$@"; do
    if [ "$skip_next" -eq 1 ]; then
        skip_next=0
        continue
    fi
    case "$arg" in
        --config)
            skip_next=1
            ;;
        --config=* | --json | --json=* | --no-color | --no-color=* | --verbose | --verbose=* | -v)
            ;;
        -*)
            ;;
        *)
            command_name=$arg
            break
            ;;
    esac
done

# Workers and one-shot commands do not need the JavaScript runtime. They reuse
# the exact same image and binary as the HTTP pod without starting the renderer.
if [ "$command_name" != "serve" ]; then
    exec /usr/local/bin/shrike "$@"
fi

NUXT_SOCKET="${NUXT_SOCKET:-/tmp/shrike-nuxt.sock}"
case "$NUXT_SOCKET" in
    /*) ;;
    *)
        echo "NUXT_SOCKET must be an absolute path" >&2
        exit 1
        ;;
esac
if [ "$NUXT_SOCKET" = "/" ]; then
    echo "NUXT_SOCKET cannot be /" >&2
    exit 1
fi

PORT="${PORT:-4000}"
NUXT_API_ORIGIN="${NUXT_API_ORIGIN:-http://127.0.0.1:${PORT}}"
NITRO_UNIX_SOCKET="$NUXT_SOCKET"
export PORT NUXT_SOCKET NUXT_API_ORIGIN NITRO_UNIX_SOCKET

rm -f -- "$NUXT_SOCKET"

renderer_pid=
server_pid=
stopping=0

terminate() {
    trap - INT TERM HUP
    stopping=1
    if [ -n "$server_pid" ]; then
        kill -TERM "$server_pid" 2>/dev/null || true
    fi
    if [ -n "$renderer_pid" ]; then
        kill -TERM "$renderer_pid" 2>/dev/null || true
    fi
}

cleanup() {
    rm -f -- "$NUXT_SOCKET"
}

trap terminate INT TERM HUP
trap cleanup EXIT

node /app/web/server/index.mjs &
renderer_pid=$!

# Caddy should not accept frontend traffic until Nitro has actually bound its
# socket. This also turns an immediate renderer crash into a pod startup
# failure instead of a live server returning 502 for every page.
attempt=0
while [ ! -S "$NUXT_SOCKET" ]; do
    if ! kill -0 "$renderer_pid" 2>/dev/null; then
        wait "$renderer_pid"
        status=$?
        if [ "$status" -eq 0 ]; then
            status=1
        fi
        echo "Nuxt renderer exited before binding ${NUXT_SOCKET}" >&2
        exit "$status"
    fi
    attempt=$((attempt + 1))
    if [ "$attempt" -ge 300 ]; then
        echo "Nuxt renderer did not bind ${NUXT_SOCKET} within 30 seconds" >&2
        terminate
        wait "$renderer_pid" 2>/dev/null || true
        exit 1
    fi
    sleep 0.1
done

/usr/local/bin/shrike "$@" &
server_pid=$!

# BusyBox sh has no portable wait-for-either primitive. Polling keeps the
# supervisor small while still making either child authoritative: if Caddy or
# Nitro dies, the other is terminated and the pod exits with the first
# process's status.
while kill -0 "$server_pid" 2>/dev/null && kill -0 "$renderer_pid" 2>/dev/null; do
    sleep 1
done

if [ "$stopping" -eq 1 ]; then
    wait "$server_pid"
    status=$?
    wait "$renderer_pid" 2>/dev/null || true
elif ! kill -0 "$server_pid" 2>/dev/null; then
    wait "$server_pid"
    status=$?
    kill -TERM "$renderer_pid" 2>/dev/null || true
    wait "$renderer_pid" 2>/dev/null || true
else
    wait "$renderer_pid"
    status=$?
    if [ "$status" -eq 0 ]; then
        status=1
    fi
    echo "Nuxt renderer stopped; shutting down Shrike" >&2
    kill -TERM "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
fi

exit "$status"
