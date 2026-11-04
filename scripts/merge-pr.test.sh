#!/usr/bin/env bash
# Checks scripts/merge-pr.sh against a fake `gh`, so the decision to arm
# auto-merge, merge now, or fail is tested without GitHub. Run from the
# repo root.
set -euo pipefail

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

UNSTABLE="GraphQL: Pull request Pull request is in unstable status (enablePullRequestAutoMerge)"
CLEAN="GraphQL: Pull request Pull request is in clean status (enablePullRequestAutoMerge)"
TRANSIENT="GraphQL: Something went wrong while executing your query. Please include 7809:31BA5D when reporting this issue."
BADGATEWAY='non-200 OK status code: 502 Bad Gateway body: "<html>"'
CONFLICT="X Pull request is not mergeable: the merge commit cannot be cleanly created."

# A fake gh that logs its arguments and answers from the environment:
# FAKE_AUTO_* for `pr merge --auto`, FAKE_CHECKS_RC for `pr checks`,
# FAKE_DIRECT_* for a plain `pr merge`. FAKE_TRANSIENT_FIRST=N makes the
# first N `--auto` calls fail with a transient GitHub error before it
# answers as FAKE_AUTO_* says.
cat >"$tmp/gh" <<'GH'
#!/usr/bin/env bash
echo "$*" >>"$FAKE_LOG"
if [ "$1 $2" = "pr checks" ]; then exit "$FAKE_CHECKS_RC"; fi
for arg in "$@"; do
  if [ "$arg" = "--auto" ]; then
    n=$(grep -c -- '--auto' "$FAKE_LOG")
    if [ "$n" -le "${FAKE_TRANSIENT_FIRST:-0}" ]; then
      printf '%s\n' "$FAKE_TRANSIENT_OUT"
      exit 1
    fi
    printf '%s\n' "$FAKE_AUTO_OUT"
    exit "$FAKE_AUTO_RC"
  fi
done
printf '%s\n' "$FAKE_DIRECT_OUT"
exit "$FAKE_DIRECT_RC"
GH
chmod +x "$tmp/gh"

URL=https://github.com/alrayyes/forge-dashboard-sdk-go/pull/1
failures=0

# run <name> <auto rc> <auto out> <checks rc> <direct rc> <want rc> <want calls (| separated)> [want output]
run() {
  local name=$1 auto_rc=$2 auto_out=$3 checks_rc=$4 direct_rc=$5 want_rc=$6 want_calls=$7 want_out=${8:-}
  : >"$tmp/log"
  local rc=0 output
  output=$(PATH="$tmp:$PATH" PR_URL=$URL GH_TOKEN=secret-token FAKE_LOG="$tmp/log" \
    FAKE_AUTO_RC=$auto_rc FAKE_AUTO_OUT=$auto_out FAKE_CHECKS_RC=$checks_rc \
    FAKE_DIRECT_RC=$direct_rc FAKE_DIRECT_OUT="direct refused" \
    RETRY_DELAY=0 FAKE_TRANSIENT_FIRST=${TRANSIENT_FIRST:-0} FAKE_TRANSIENT_OUT=${TRANSIENT_OUT:-} \
    ./scripts/merge-pr.sh 2>&1) || rc=$?
  local calls
  calls=$(paste -sd'|' "$tmp/log")
  if [ "$rc" != "$want_rc" ] && { [ "$want_rc" = 0 ] || [ "$rc" = 0 ]; }; then
    echo "FAIL $name: exit $rc, want $want_rc" >&2; failures=$((failures + 1)); return
  fi
  if [ "$calls" != "$want_calls" ]; then
    echo "FAIL $name: calls '$calls', want '$want_calls'" >&2; failures=$((failures + 1)); return
  fi
  if [ -n "$want_out" ] && ! grep -q "$want_out" <<<"$output"; then
    echo "FAIL $name: output lacks '$want_out'" >&2; failures=$((failures + 1)); return
  fi
  if grep -q secret-token <<<"$output"; then
    echo "FAIL $name: token printed" >&2; failures=$((failures + 1)); return
  fi
  echo "ok $name"
}

AUTO="pr merge --auto --squash $URL"
CHECKS="pr checks $URL --required"
DIRECT="pr merge --squash $URL"

run "arms auto-merge when GitHub accepts it" 0 "" 0 0 0 "$AUTO"
run "merges now when unstable and required checks pass" 1 "$UNSTABLE" 0 0 0 "$AUTO|$CHECKS|$DIRECT"
run "merges now when already clean" 1 "$CLEAN" 0 0 0 "$AUTO|$CHECKS|$DIRECT"
run "fails and merges nothing on any other refusal" 1 "$CONFLICT" 0 0 1 "$AUTO" "cannot be cleanly created"
run "does not merge when a required check is pending or failing" 1 "$UNSTABLE" 1 0 1 "$AUTO|$CHECKS"
run "fails when the direct merge is refused" 1 "$UNSTABLE" 0 1 1 "$AUTO|$CHECKS|$DIRECT" "direct refused"

# A transient GitHub error is retried; a real refusal is not.
TRANSIENT_OUT=$TRANSIENT TRANSIENT_FIRST=2 run "retries a transient GraphQL error, then arms" 0 "" 0 0 0 "$AUTO|$AUTO|$AUTO"
TRANSIENT_OUT=$BADGATEWAY TRANSIENT_FIRST=1 run "retries a 502, then arms" 0 "" 0 0 0 "$AUTO|$AUTO"
TRANSIENT_OUT=$TRANSIENT TRANSIENT_FIRST=9 run "gives up after repeated transient errors" 0 "" 0 0 1 "$AUTO|$AUTO|$AUTO|$AUTO" "Something went wrong"
TRANSIENT_OUT=$TRANSIENT TRANSIENT_FIRST=1 run "retries, then merges now when unstable" 1 "$UNSTABLE" 0 0 0 "$AUTO|$AUTO|$CHECKS|$DIRECT"

[ "$failures" = 0 ] || exit 1
echo "ok"
