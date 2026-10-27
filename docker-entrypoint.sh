#!/bin/sh
set -e

case "$1" in
  generate)
    shift
    exec spectogram "$@"
    ;;
  serve)
    shift
    exec spectogram serve "$@"
    ;;
  *)
    exec "$@"
    ;;
esac
