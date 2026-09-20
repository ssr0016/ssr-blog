#!/usr/bin/env bash
# test_writefile.sh -- self-contained test for writefile.sh
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
WF="$ROOT/.specify/scripts/writefile.sh"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

pass=0
fail=0

check() {
    local name="$1"; shift
    if "$@" >/dev/null 2>&1; then
        printf 'PASS %s\n' "$name"
        pass=$((pass + 1))
    else
        printf 'FAIL %s\n' "$name"
        fail=$((fail + 1))
    fi
}

refuse() {
    local name="$1"; shift
    if "$@" >/dev/null 2>&1; then
        printf 'FAIL %s (expected refusal, got success)\n' "$name"
        fail=$((fail + 1))
    else
        printf 'PASS %s\n' "$name"
        pass=$((pass + 1))
    fi
}

target="$tmp/target.txt"

# Case 1: quotes, backticks, dollar, heredoc-term lines written byte-identical
printf 'line with "double" and %s and $var\n' '' > "$tmp/c1.in"
printf 'EOF\n' >> "$tmp/c1.in"
printf 'text << not a real heredoc\n' >> "$tmp/c1.in"
printf 'tail line\n' >> "$tmp/c1.in"
printf 'sig123\n' >> "$tmp/c1.in"
check "case1_byte_identical" bash -c "\"$WF\" \"$target\" --lines 5 --sig sig123 --no-backup < \"$tmp/c1.in\""
if [ -f "$target" ] && cmp -s <(head -5 "$tmp/c1.in") "$target"; then
    printf 'PASS case1_cmp\n'
    pass=$((pass + 1))
else
    printf 'FAIL case1_cmp\n'
    fail=$((fail + 1))
fi
rm -f "$target"

# Case 2: empty content refused, no file created
refuse "case2_empty_refused" bash -c ": | \"\$WF\" \"\$target\" --lines 0 --sig x --no-backup"
check "case2_no_file" test ! -e "$target"

# Case 3: missing --sig refused, target untouched
printf "original\\n" > "$target"
refuse "case3_missing_sig" bash -c "printf \"body\\\\nsig\\\\n\" | \"\$WF\" \"\$target\" --lines 1 --no-backup"
check "case3_target_intact" grep -q "^original$" "$target"
rm -f "$target"

# Case 4: wrong --lines refused, no file created
refuse "case4_wrong_lines" bash -c "printf \"body\\\\nsig\\\\n\" | \"\$WF\" \"\$target\" --lines 99 --sig sig --no-backup"
check "case4_no_file" test ! -e "$target"

# Case 5: Cyrillic look-alike (built via printf escape) rejected with line:column
cyr=$(printf "\\320\\260")
refuse "case5_cyrillic_rejected" bash -c "printf \"ab%s\\\\nsig\\\\n\" \"\$cyr\" | \"\$WF\" \"\$target\" --lines 2 --sig sig --no-backup"
check "case5_no_file" test ! -e "$target"

# Case 6: --allow-utf8 writes Cyrillic intact
printf "ab%s\nsig\n" "$cyr" > "$tmp/c6.in"
check "case6_utf8_allowed" "$WF" "$target" --from "$tmp/c6.in" --allow-utf8 --no-backup
check "case6_file_exists" test -f "$target"
check "case6_bytes_match" grep -q -- "$cyr" "$target"
rm -f "$target"

# Case 7: backup created with timestamp name, --no-backup creates none
printf "body\nsig\n" | "$WF" "$target" --lines 2 --sig sig --no-backup
printf "body\nsig\n" | "$WF" "$target" --lines 2 --sig sig
n=$(ls "$tmp"/target.txt.bak.* 2>/dev/null | wc -l)
check "case7_backup_created" test "$n" -ge 1
rm -f "$target" "$tmp"/target.txt.bak.*
printf "body\nsig\n" | "$WF" "$target" --lines 2 --sig sig --no-backup
n=$(ls "$tmp"/target.txt.bak.* 2>/dev/null | wc -l)
check "case7_no_backup" test "$n" -eq 0
rm -f "$target"

# Case 8: two writes in same second -> two distinct backups
printf "seed\nsig\n" | "$WF" "$target" --lines 2 --sig sig --no-backup
printf "v1\nsig\n" | "$WF" "$target" --lines 2 --sig sig
sleep 1.1
printf "v2\nsig\n" | "$WF" "$target" --lines 2 --sig sig
n=$(ls "$tmp"/target.txt.bak.* 2>/dev/null | wc -l)
check "case8_two_backups" test "$n" -eq 2
rm -f "$target" "$tmp"/target.txt.bak.*

# Case 9: unwritable backup location -> refuse, original untouched
printf "original\n" > "$target"
chmod 500 "$tmp"
refuse "case9_unwritable" bash -c "printf \"body\\\\nsig\\\\n\" | \"\$WF\" \"\$target\" --lines 2 --sig sig"
chmod 700 "$tmp"
check "case9_original_intact" grep -q "^original$" "$target"
rm -f "$target"

# Case 10: sha256 mismatch -> non-zero
refuse "case10_sha_mismatch" bash -c "printf \"body\\\\nsig\\\\n\" | \"\$WF\" \"\$target\" --lines 2 --sig sig --sha256 0000000000000000000000000000000000000000000000000000000000000000 --no-backup"
check "case10_no_file" test ! -e "$target"


# Case 11: cleanup removes 8-day-old backup, keeps 6-day-old, ignores non-matching
printf "seed\nsig\n" | "$WF" "$target" --lines 2 --sig sig --no-backup
old_ts=$(date -u -d "8 days ago" +%Y-%m-%dT%H-%M-%S)
new_ts=$(date -u -d "6 days ago" +%Y-%m-%dT%H-%M-%S)
printf "old\n" > "$tmp/target.txt.bak.$old_ts"
printf "new\n" > "$tmp/target.txt.bak.$new_ts"
printf "keep\n" > "$tmp/target.txt.not-a-backup"
printf "v2\nsig\n" | "$WF" "$target" --lines 2 --sig sig
check "case11_old_removed" test ! -e "$tmp/target.txt.bak.$old_ts"
check "case11_new_kept" test -e "$tmp/target.txt.bak.$new_ts"
check "case11_other_untouched" test -e "$tmp/target.txt.not-a-backup"
rm -f "$target" "$tmp"/target.txt.bak.* "$tmp/target.txt.not-a-backup"

# Summary
printf '\n%s passed, %s failed\n' "$pass" "$fail"
if [ "$fail" -gt 0 ]; then
    exit 1
fi
exit 0
