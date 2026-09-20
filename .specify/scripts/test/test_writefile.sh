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
    "$@" >/dev/null 2>&1
    local rc=$?
    if [ "$rc" -eq 0 ]; then
        printf 'FAIL %s (expected refusal, got success)\n' "$name"
        fail=$((fail + 1))
    elif [ "$rc" -eq 126 ] || [ "$rc" -eq 127 ]; then
        # The command itself could not run: that is not a refusal by writefile.sh.
        printf 'FAIL %s (command could not run, exit %s)\n' "$name" "$rc"
        fail=$((fail + 1))
    else
        printf 'PASS %s\n' "$name"
        pass=$((pass + 1))
    fi
}

target="$tmp/target.txt"

# The refuse cases below run writefile.sh inside "bash -c", which only sees exported
# variables. Without this export the inner command is empty, every refusal passes
# trivially, and nothing about writefile.sh is tested.
export WF tmp target

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

# Case 3: in stdin mode both --lines and --sig are required; a missing one is refused
# even when the other is correct, and the target is untouched
printf "original\\n" > "$target"
refuse "case3_missing_sig" bash -c "printf \"body\\\\nsig\\\\n\" | \"\$WF\" \"\$target\" --lines 2 --no-backup"
check "case3_target_intact" grep -q "^original$" "$target"
refuse "case3_missing_lines" bash -c "printf \"body\\\\nsig\\\\n\" | \"\$WF\" \"\$target\" --sig sig --no-backup"
check "case3_missing_lines_target_intact" grep -q "^original$" "$target"
refuse "case3_neither_flag" bash -c "printf \"body\\\\nsig\\\\n\" | \"\$WF\" \"\$target\" --no-backup"
check "case3_neither_flag_target_intact" grep -q "^original$" "$target"
refuse "case3_empty_sig_value" bash -c "printf \"body\\\\nsig\\\\n\" | \"\$WF\" \"\$target\" --lines 2 --sig \"\" --no-backup"
check "case3_no_stray_temp" bash -c '! ls -A "$1" | grep -q "^\.writefile\."' _ "$tmp"
rm -f "$target"

# Case 4: wrong --lines refused, no file created
refuse "case4_wrong_lines" bash -c "printf \"body\\\\nsig\\\\n\" | \"\$WF\" \"\$target\" --lines 99 --sig sig --no-backup"
check "case4_no_file" test ! -e "$target"

# Case 5: Cyrillic look-alike (built via printf escape) rejected with line:column
cyr=$(printf "\\320\\260")
export cyr
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

# Case 12: a directory as TARGET is refused in both backup modes, with nothing left behind
mkdir -p "$tmp/adir"
refuse "case12_dir_target_no_backup" bash -c "printf \"body\\\\nsig\\\\n\" | \"\$WF\" \"\$tmp/adir\" --lines 2 --sig sig --no-backup"
refuse "case12_dir_target_with_backup" bash -c "printf \"body\\\\nsig\\\\n\" | \"\$WF\" \"\$tmp/adir\" --lines 2 --sig sig"
check "case12_dir_stays_empty" test -z "$(ls -A "$tmp/adir")"
check "case12_no_stray_temp" bash -c '! ls -A "$1" | grep -q "^\.writefile\."' _ "$tmp"
rmdir "$tmp/adir"

# Case 13: the 7-day cleanup covers the whole target directory on every write:
# other files' backups, a new target, and --no-backup. Names that do not match
# the strict pattern are never touched.
old_ts=$(date -u -d "8 days ago" +%Y-%m-%dT%H-%M-%S)
new_ts=$(date -u -d "6 days ago" +%Y-%m-%dT%H-%M-%S)
seed_others() {
    printf "old\n" > "$tmp/other.txt.bak.$old_ts"
    printf "new\n" > "$tmp/other.txt.bak.$new_ts"
    printf "keep\n" > "$tmp/other.txt.not-a-backup"
    printf "keep\n" > "$tmp/other.txt.bak.notatimestamp"
    # Names that look like old backups but do not match the strict pattern. They sort
    # before the cutoff, so only the pattern (not the age comparison) protects them.
    printf "keep\n" > "$tmp/other.txt.bak.2000-01-01"
    printf "keep\n" > "$tmp/other.txt.bak.2000-01-01T00-00-00x"
    printf "keep\n" > "$tmp/other.txt.bak.1999-12-31T23-59-59.gz"
}
clear_others() { rm -f "$tmp"/other.txt.* "$target" "$tmp"/target.txt.bak.*; }

seed_others
printf "v1\nsig\n" | "$WF" "$target" --lines 2 --sig sig
check "case13_new_target_prunes_other_old" test ! -e "$tmp/other.txt.bak.$old_ts"
check "case13_new_target_keeps_recent" test -e "$tmp/other.txt.bak.$new_ts"
check "case13_new_target_keeps_non_matching" test -e "$tmp/other.txt.not-a-backup"
check "case13_new_target_keeps_bad_timestamp" test -e "$tmp/other.txt.bak.notatimestamp"
check "case13_new_target_keeps_date_only_name" test -e "$tmp/other.txt.bak.2000-01-01"
check "case13_new_target_keeps_trailing_junk_name" test -e "$tmp/other.txt.bak.2000-01-01T00-00-00x"
check "case13_new_target_keeps_suffixed_name" test -e "$tmp/other.txt.bak.1999-12-31T23-59-59.gz"
check "case13_new_target_written" grep -q "^v1$" "$target"
clear_others

printf "seed\nsig\n" | "$WF" "$target" --lines 2 --sig sig --no-backup
seed_others
printf "v2\nsig\n" | "$WF" "$target" --lines 2 --sig sig --no-backup
check "case13_no_backup_write_still_prunes" test ! -e "$tmp/other.txt.bak.$old_ts"
check "case13_no_backup_makes_no_backup" bash -c '! ls "$1"/target.txt.bak.* >/dev/null 2>&1' _ "$tmp"
clear_others

# A directory named like a backup is left alone and does not fail the write
printf "seed\nsig\n" | "$WF" "$target" --lines 2 --sig sig --no-backup
mkdir "$tmp/stuck.bak.$old_ts"
printf "v3\nsig\n" | "$WF" "$target" --lines 2 --sig sig --no-backup
rc=$?
check "case13_dir_named_like_backup_write_exit_zero" test "$rc" -eq 0
check "case13_dir_named_like_backup_write_landed" grep -q "^v3$" "$target"
check "case13_dir_named_like_backup_left_alone" test -d "$tmp/stuck.bak.$old_ts"
rmdir "$tmp/stuck.bak.$old_ts"
clear_others

# Case 14: dotfile backups (names that start with a dot) are pruned too, both the
# backups of a dotfile target and the backups of a dotfile next to an ordinary target
printf "seed\nsig\n" | "$WF" "$tmp/.dotfile" --lines 2 --sig sig --no-backup
printf "old\n" > "$tmp/.dotfile.bak.$old_ts"
printf "new\n" > "$tmp/.dotfile.bak.$new_ts"
printf "keep\n" > "$tmp/.dotfile.not-a-backup"
printf "v2\nsig\n" | "$WF" "$tmp/.dotfile" --lines 2 --sig sig
check "case14_dotfile_old_backup_pruned" test ! -e "$tmp/.dotfile.bak.$old_ts"
check "case14_dotfile_recent_backup_kept" test -e "$tmp/.dotfile.bak.$new_ts"
check "case14_dotfile_non_backup_kept" test -e "$tmp/.dotfile.not-a-backup"
check "case14_dotfile_write_landed" grep -q "^v2$" "$tmp/.dotfile"
printf "old\n" > "$tmp/.other.bak.$old_ts"
printf "v1\nsig\n" | "$WF" "$target" --lines 2 --sig sig --no-backup
check "case14_dotfile_backup_beside_ordinary_target_pruned" test ! -e "$tmp/.other.bak.$old_ts"
rm -f "$tmp"/.dotfile "$tmp"/.dotfile.* "$tmp"/.other.bak.* "$target"

# Case 15: a real cleanup failure must not fail or undo the write. A stub rm that always
# fails stands in for an undeletable backup, and its log proves the deletion was tried.
mkdir -p "$tmp/stubbin"
{ printf '%s\n' '#!/bin/sh' 'printf "%s\n" "$*" >> "$RM_LOG"' 'exit 1'; } > "$tmp/stubbin/rm"
chmod +x "$tmp/stubbin/rm"
export RM_LOG="$tmp/rm.log"
: > "$RM_LOG"
printf "seed\nsig\n" | "$WF" "$target" --lines 2 --sig sig --no-backup
printf "old\n" > "$tmp/other.txt.bak.$old_ts"
printf "v5\nsig\n" | PATH="$tmp/stubbin:$PATH" "$WF" "$target" --lines 2 --sig sig --no-backup
rc=$?
check "case15_write_exit_zero_when_rm_fails" test "$rc" -eq 0
check "case15_write_landed_when_rm_fails" grep -q "^v5$" "$target"
check "case15_rm_was_tried_on_the_old_backup" grep -q -F "other.txt.bak.$old_ts" "$RM_LOG"
check "case15_undeletable_backup_still_there" test -e "$tmp/other.txt.bak.$old_ts"
check "case15_stub_is_first_on_path" bash -c 'PATH="$1:$PATH"; [ "$(command -v rm)" = "$1/rm" ]' _ "$tmp/stubbin"
rm -rf "$tmp/stubbin" "$RM_LOG"
clear_others

# Summary
printf '\n%s passed, %s failed\n' "$pass" "$fail"
if [ "$fail" -gt 0 ]; then
    exit 1
fi
exit 0
