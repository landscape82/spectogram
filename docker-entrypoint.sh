#!/bin/sh
set -e

case "$1" in
  generate)
    shift
    exec spectogram "$@"
    ;;
  serve)
    exec python3 -m http.server 8000
    ;;
  *)
    exec "$@"
    ;;
esac
