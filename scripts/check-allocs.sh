#!/bin/sh
# Fails when any reggol benchmark reports a non-zero allocs/op.
#
# This is the blocking performance gate. Unlike ns/op, which swings threefold on
# a shared runner, allocation counts are deterministic — so they are what CI can
# actually enforce.
#
# BenchmarkStdlibSlogText is excluded: it measures the standard library as a
# reference point, and its allocations are not ours to fix.
#
# benchtime must stay long enough for the one-time sync.Pool warmup to amortize;
# a handful of iterations reports those startup allocations as steady state.
set -eu

out=$(go test -run '^$' -bench=. -benchmem -benchtime=200ms "${1:-./...}" 2>&1)

echo "$out"

bad=$(printf '%s\n' "$out" \
    | grep -E '^Benchmark' \
    | grep -v 'BenchmarkStdlibSlogText' \
    | awk '$NF == "allocs/op" && $(NF-1) != "0" { print }')

if [ -n "$bad" ]; then
    echo
    echo "FAIL: benchmarks allocating on the hot path:"
    printf '%s\n' "$bad"
    exit 1
fi

echo
echo "OK: every reggol benchmark reports 0 allocs/op"
