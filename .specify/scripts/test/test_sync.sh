#!/usr/bin/env bash
# test_sync.sh -- self-contained test for sync.sh
# Every sync.sh call runs with HOME and SPEC_KIT_HOME pointed into a temp dir,
# so the real ~/.spec-kit is never written. The last cases prove that by
# comparing a snapshot of the real home taken before and after the run.
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
SYNC="$ROOT/.specify/scripts/sync.sh"
REAL_HOME="$HOME"
FILES="writefile.sh verify-edit.sh sync.sh converge.yaml"

tmp="$(mktemp -d)"
trap 'chmod -R u+rwX "$tmp" 2>/dev/null; rm -rf "$tmp"' EXIT

pass=0
fail=0

ok()  { printf 'PASS %s\n' "$1"; pass=$((pass + 1)); }
bad() { printf 'FAIL %s\n' "$1"; fail=$((fail + 1)); }

expect_exit() {
    local name="$1"; local want="$2"; shift 2
    "$@" >/dev/null 2>&1
    local got=$?
    if [ "$got" -eq "$want" ]; then ok "$name"; else bad "$name (want exit $want, got $got)"; fi
}

assert() {
    local name="$1"; shift
    if "$@" >/dev/null 2>&1; then ok "$name"; else bad "$name"; fi
}

# sync_in HOMEDIR ARGS... : run sync.sh with SPEC_KIT_HOME=HOMEDIR, sandboxed HOME.
sync_in() {
    local h="$1"; shift
    HOME="$tmp/sandbox-home" SPEC_KIT_HOME="$h" "$SYNC" "$@"
}

# sync_default HOMEDIR ARGS... : SPEC_KIT_HOME unset, HOME=HOMEDIR.
sync_default() {
    local h="$1"; shift
    env -u SPEC_KIT_HOME HOME="$h" "$SYNC" "$@"
}

sha() { sha256sum "$1" | awk '{print $1}'; }

snapshot_real_home() {
    if [ -d "$REAL_HOME/.spec-kit" ]; then
        (cd "$REAL_HOME/.spec-kit" && find . -type f -exec sha256sum {} + | sort)
    else
        echo ABSENT
    fi
}
real_before="$(snapshot_real_home)"

mkdir -p "$tmp/sandbox-home"

# Case 0: the script exists and parses
assert "case0_exists" test -f "$SYNC"
assert "case0_bash_n" bash -n "$SYNC"

# Case 1: install creates a missing home dir (nested), then check passes
h1="$tmp/h1/nested/deeper"
expect_exit "case1_install_creates_dir" 0 sync_in "$h1" install
for f in $FILES; do assert "case1_has_$f" test -f "$h1/$f"; done
expect_exit "case1_check_identical" 0 sync_in "$h1" check
out="$(sync_in "$h1" check 2>&1)"
if printf '%s\n' "$out" | grep -q -E 'missing|differ'; then
    bad "case1_check_output_clean"
else
    ok "case1_check_output_clean"
fi

# Case 2: check on a missing home reports each file, exits 1, creates nothing
h2="$tmp/h2"
expect_exit "case2_check_missing_exit" 1 sync_in "$h2" check
out="$(sync_in "$h2" check 2>&1)"
for f in $FILES; do
    if printf '%s\n' "$out" | grep "$f" | grep -q 'missing'; then
        ok "case2_reports_missing_$f"
    else
        bad "case2_reports_missing_$f"
    fi
done
assert "case2_check_creates_nothing" test ! -e "$h2"

# Case 3: a differing home copy is named, exit is 1, file unchanged
printf 'local edit\n' >> "$h1/writefile.sh"
before="$(sha "$h1/writefile.sh")"
expect_exit "case3_check_differ_exit" 1 sync_in "$h1" check
out="$(sync_in "$h1" check 2>&1)"
if printf '%s\n' "$out" | grep 'writefile.sh' | grep -q 'differ'; then
    ok "case3_names_differing_file"
else
    bad "case3_names_differing_file"
fi
if [ "$(sha "$h1/writefile.sh")" = "$before" ]; then ok "case3_file_unchanged"; else bad "case3_file_unchanged"; fi

# Case 4: install refuses to overwrite a differing file without --force
expect_exit "case4_install_refuses" 1 sync_in "$h1" install
out="$(sync_in "$h1" install 2>&1)"
if printf '%s\n' "$out" | grep -q 'writefile.sh'; then ok "case4_names_file"; else bad "case4_names_file"; fi
if [ "$(sha "$h1/writefile.sh")" = "$before" ]; then ok "case4_file_unchanged"; else bad "case4_file_unchanged"; fi

# Case 5: refusal is all-or-nothing (missing files are not installed either)
h5="$tmp/h5"
mkdir -p "$h5"
printf 'stale settings\n' > "$h5/converge.yaml"
stale="$(sha "$h5/converge.yaml")"
expect_exit "case5_install_refuses" 1 sync_in "$h5" install
for f in writefile.sh verify-edit.sh sync.sh; do
    assert "case5_not_installed_$f" test ! -e "$h5/$f"
done
if [ "$(sha "$h5/converge.yaml")" = "$stale" ]; then ok "case5_converge_unchanged"; else bad "case5_converge_unchanged"; fi

# Case 6: --force overwrites, backs up only the differing file, check passes
expect_exit "case6_force_install" 0 sync_in "$h5" install --force
expect_exit "case6_check_after_force" 0 sync_in "$h5" check
nbak="$(find "$h5" -maxdepth 1 -name '*.bak.*' | wc -l)"
if [ "$nbak" -eq 1 ]; then ok "case6_one_backup"; else bad "case6_one_backup (found $nbak)"; fi
bakfile="$(find "$h5" -maxdepth 1 -name 'converge.yaml.bak.*' | head -n 1)"
if printf '%s\n' "$bakfile" | grep -q -E '/converge\.yaml\.bak\.[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}-[0-9]{2}-[0-9]{2}$'; then
    ok "case6_backup_name_pattern"
else
    bad "case6_backup_name_pattern ($bakfile)"
fi
if [ -n "$bakfile" ] && [ "$(sha "$bakfile")" = "$stale" ]; then ok "case6_backup_has_old_content"; else bad "case6_backup_has_old_content"; fi
expect_exit "case6_force_on_identical" 0 sync_in "$h5" install --force
nbak="$(find "$h5" -maxdepth 1 -name '*.bak.*' | wc -l)"
if [ "$nbak" -eq 1 ]; then ok "case6_identical_makes_no_backup"; else bad "case6_identical_makes_no_backup (found $nbak)"; fi

# Case 6b: plain install on identical copies succeeds and changes nothing
expect_exit "case6b_install_identical" 0 sync_in "$h5" install

# Case 7: usage errors exit 64
expect_exit "case7_no_mode" 64 sync_in "$tmp/h7"
expect_exit "case7_unknown_mode" 64 sync_in "$tmp/h7" bogus
expect_exit "case7_unknown_flag" 64 sync_in "$tmp/h7" install --bogus
expect_exit "case7_force_on_check" 64 sync_in "$tmp/h7" check --force
assert "case7_nothing_created" test ! -e "$tmp/h7"

# Case 7b: an unreadable home copy exits 2
h7b="$tmp/h7b"
sync_in "$h7b" install >/dev/null 2>&1
chmod 000 "$h7b/converge.yaml"
expect_exit "case7b_unreadable" 2 sync_in "$h7b" check
chmod 644 "$h7b/converge.yaml"

# Case 8: SPEC_KIT_HOME unset -> default is $HOME/.spec-kit
h8="$tmp/fakehome8"
mkdir -p "$h8"
expect_exit "case8_default_home_install" 0 sync_default "$h8" install
assert "case8_default_home_used" test -f "$h8/.spec-kit/converge.yaml"
expect_exit "case8_default_home_check" 0 sync_default "$h8" check

# Case 8b: SPEC_KIT_HOME set explicitly -> that path is used, default home untouched
h8b="$tmp/fakehome8b"
c8b="$tmp/custom8b"
mkdir -p "$h8b"
expect_exit "case8b_custom_install" 0 env HOME="$h8b" SPEC_KIT_HOME="$c8b" "$SYNC" install
for f in $FILES; do assert "case8b_custom_has_$f" test -f "$c8b/$f"; done
assert "case8b_default_untouched" test ! -e "$h8b/.spec-kit"
expect_exit "case8b_custom_check" 0 env HOME="$h8b" SPEC_KIT_HOME="$c8b" "$SYNC" check

# Case 9: static checks on sync.sh and on this test file
assert "case9_sync_ascii" bash -c "! LC_ALL=C grep -q -P '[^\\x00-\\x7F]' \"$SYNC\""
assert "case9_test_ascii" bash -c "! LC_ALL=C grep -q -P '[^\\x00-\\x7F]' \"$0\""
marker="$(printf '<%s' '<')"
if [ -f "$SYNC" ] && [ "$(grep -c -F -e "$marker" "$SYNC")" -eq 0 ]; then ok "case9_sync_no_heredoc"; else bad "case9_sync_no_heredoc"; fi
if [ "$(grep -c -F -e "$marker" "$0")" -eq 0 ]; then ok "case9_test_no_heredoc"; else bad "case9_test_no_heredoc"; fi

# Case 10: the real home was never touched by any case above
real_after="$(snapshot_real_home)"
if [ "$real_before" = "$real_after" ]; then ok "case10_real_home_untouched"; else bad "case10_real_home_untouched"; fi

printf '\n%s passed, %s failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
