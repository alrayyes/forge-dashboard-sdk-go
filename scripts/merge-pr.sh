#!/usr/bin/env bash
# Arms auto-merge on the pull request in $PR_URL, or merges it now when
# there is nothing left to wait for.
#
# `gh pr merge --auto` asks GitHub to merge once the required checks pass.
# GitHub refuses that, with "in unstable status", when the pull request is
# `unstable`: every required check already passed and only a non-required
# one is still going or failed, which is the state an auto-merge job lands
# in when it starts a minute or two after the pull request opens. "In clean
# status" is the same answer for a pull request that is already mergeable.
# Both mean there is nothing left to wait for, so merge now. Any other
# refusal (a merge conflict, a required check still pending) is a real
# failure and stops the job with GitHub's own message.
#
# This repo's main does not enforce rules on admins, so GitHub would let an
# admin token merge past a pending or failing required check. Before the
# direct merge, `gh pr checks --required` must pass: it exits non-zero while
# any required check is pending or failing.
# https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/automatically-merging-a-pull-request
#
# GitHub's API answers a 5xx or "Something went wrong" now and then (a 502
# and a GraphQL error both turned up on #417). Those say nothing about the
# pull request, so a call that gets one is retried (RETRY_DELAY seconds
# apart, RETRIES times) before it counts as a failure. Any other refusal
# fails at once.
set -euo pipefail

: "${PR_URL:?PR_URL is required}"
RETRIES=${RETRIES:-4}
RETRY_DELAY=${RETRY_DELAY:-15}

# gh_retry <gh args...>: runs gh, sets $out to its output and returns its
# status, retrying only when the output looks like a transient GitHub error.
gh_retry() {
  local attempt=1 rc
  while :; do
    rc=0
    out=$(gh "$@" 2>&1) || rc=$?
    [ "$rc" = 0 ] && return 0
    if ! grep -qiE 'something went wrong|server error|bad gateway|service unavailable|gateway time-?out|(HTTP|status code:?) 50[0-4]' <<<"$out" \
      || [ "$attempt" -ge "$RETRIES" ]; then
      return "$rc"
    fi
    echo "transient GitHub error (attempt $attempt/$RETRIES), retrying: $out" >&2
    sleep "$RETRY_DELAY"
    attempt=$((attempt + 1))
  done
}

if gh_retry pr merge --auto --squash "$PR_URL"; then
  echo "$out"
  exit 0
fi
echo "$out" >&2

case "$out" in
  *"is in unstable status"* | *"is in clean status"*)
    gh pr checks "$PR_URL" --required
    gh_retry pr merge --squash "$PR_URL" || { echo "$out" >&2; exit 1; }
    ;;
  *)
    exit 1
    ;;
esac
