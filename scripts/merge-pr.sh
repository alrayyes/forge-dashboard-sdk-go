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
set -euo pipefail

: "${PR_URL:?PR_URL is required}"

if out=$(gh pr merge --auto --squash "$PR_URL" 2>&1); then
  echo "$out"
  exit 0
fi
echo "$out" >&2

case "$out" in
  *"is in unstable status"* | *"is in clean status"*)
    gh pr checks "$PR_URL" --required
    gh pr merge --squash "$PR_URL"
    ;;
  *)
    exit 1
    ;;
esac
