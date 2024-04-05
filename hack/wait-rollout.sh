#!/bin/bash

n=0
until oc get "$@"; do
  echo "waiting for resources: $*"
  sleep 5
  [ $(( ++n )) = 6 ] && { echo "timed out waiting for $*"; return 1; }
done

oc rollout status --watch --timeout=5m "$@" || exit 1