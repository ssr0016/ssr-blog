#!/usr/bin/env bash
# run.sh -- run every test_*.sh in this directory, summarize, exit non-zero on any failure.
set -u

HERE="$(cd "$(dirname "$0")" && pwd)"
pass=0
fail=0
failed_names=""

for t in "$HERE"/test_*.sh; do
    [ -e "$t" ] || continue
    name="$(basename "$t")"
    if bash "$t"; then
        printf 'PASS %s\n' "$name"
        pass=$((pass + 1))
    else
        printf 'FAIL %s\n' "$name"
        fail=$((fail + 1))
        failed_names="$failed_names $name"
    fi
done

printf '\n%s passed, %s failed\n' "$pass" "$fail"
if [ "$fail" -gt 0 ]; then
    printf 'failed:%s\n' "$failed_names"
    exit 1
fi
exit 0
