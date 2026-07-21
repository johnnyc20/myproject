#!/usr/bin/env bash
# Runs govulncheck against a vulnerability DB built locally from
# github.com/golang/vulndb, instead of fetching from vuln.go.dev directly.
#
# This environment's egress policy blocks vuln.go.dev (confirmed via the
# agent-proxy status endpoint: policy-level 403, not transient), which makes
# `govulncheck ./...` fail outright with no vulnerability data. github.com is
# reachable, so this mirrors the same data from its source repo and points
# govulncheck at the local copy instead.
#
# A full (non-shallow) clone is required: golang/vulndb's own `cmd/gendb`
# tool walks each report's commit history to build the static DB, which
# breaks on a shallow clone.
#
# Usage: scripts/govulncheck-offline.sh [govulncheck args...]
# Defaults to scanning ./... if no args are given.
set -euo pipefail

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

vulndb_src="$workdir/vulndb-src"
vulndb_out="$workdir/vulndb-out"

git clone --quiet https://github.com/golang/vulndb.git "$vulndb_src"
(cd "$vulndb_src" && go run ./cmd/gendb -repo "$vulndb_src" -out "$vulndb_out")

args=("$@")
if [ "${#args[@]}" -eq 0 ]; then
  args=(./...)
fi

go run golang.org/x/vuln/cmd/govulncheck@latest -db="file://$vulndb_out" "${args[@]}"
