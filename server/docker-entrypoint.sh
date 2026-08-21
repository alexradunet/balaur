#!/bin/sh
set -eu

if [ "${1:-}" = "serve" ]; then
  setup_ttl=${BALAUR_SETUP_TTL_SECONDS:-1800}
  case "$setup_ttl" in
    ''|*[!0-9]*)
      printf 'BALAUR_SETUP_TTL_SECONDS must contain a positive number.\n' >&2
      exit 64
      ;;
  esac
  if [ "$setup_ttl" -le 0 ]; then
    printf 'BALAUR_SETUP_TTL_SECONDS must contain a positive number.\n' >&2
    exit 64
  fi

  if [ -z "${BALAUR_SETUP_SECRET:-}" ]; then
    BALAUR_SETUP_SECRET=$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')
    export BALAUR_SETUP_SECRET
  fi

  case "$BALAUR_SETUP_SECRET" in
    *[!A-Za-z0-9]*)
      printf 'BALAUR_SETUP_SECRET must contain only letters and numbers.\n' >&2
      exit 64
      ;;
  esac
  if [ "${#BALAUR_SETUP_SECRET}" -lt 32 ] || [ "${#BALAUR_SETUP_SECRET}" -gt 128 ]; then
    printf 'BALAUR_SETUP_SECRET must contain 32 to 128 characters.\n' >&2
    exit 64
  fi

  if [ -z "${BALAUR_SETUP_EXPIRES_AT:-}" ]; then
    BALAUR_SETUP_EXPIRES_AT=$(($(date +%s) + setup_ttl))
    export BALAUR_SETUP_EXPIRES_AT
  fi
  case "$BALAUR_SETUP_EXPIRES_AT" in
    ''|*[!0-9]*)
      printf 'BALAUR_SETUP_EXPIRES_AT must contain a Unix timestamp.\n' >&2
      exit 64
      ;;
  esac
fi

exec /pb/pocketbase "$@"
