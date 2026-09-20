#!/usr/bin/env bash
# test_verify_edit.sh -- self-contained test for verify-edit.sh
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
VE="$ROOT/.specify/scripts/verify-edit.sh"

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
        printf 'FAIL %s (expected non-zero, got success)\n' "$name"
        fail=$((fail + 1))
    else
        printf 'PASS %s\n' "$name"
        pass=$((pass + 1))
    fi
}

expect_exit() {
    local name="$1"; local want="$2"; shift 2
    "$@" >/dev/null 2>&1
    local got=$?
    if [ "$got" -eq "$want" ]; then
        printf 'PASS %s\n' "$name"
        pass=$((pass + 1))
    else
        printf 'FAIL %s (want exit %s, got %s)\n' "$name" "$want" "$got"
        fail=$((fail + 1))
    fi
}

target="$tmp/file.txt"
printf "alpha\\nbeta\\ngamma\\n" > "$target"

# Case 1: --contains passes when present, exits 1 when absent
expect_exit "case1_contains_present" 0 "$VE" "$target" --contains beta
expect_exit "case1_contains_absent" 1 "$VE" "$target" --contains zzz

# Case 2: --absent passes when gone, exits 1 when still there
expect_exit "case2_absent_gone" 0 "$VE" "$target" --absent zzz
expect_exit "case2_absent_present" 1 "$VE" "$target" --absent beta

# Case 3: --count passes on exact count, exits 1 on different, including more
printf "aa\\nbb\\naa\\ncc\\naa\\n" > "$target"
expect_exit "case3_count_exact" 0 "$VE" "$target" --count "aa=3"
expect_exit "case3_count_low" 1 "$VE" "$target" --count "aa=2"
expect_exit "case3_count_high" 1 "$VE" "$target" --count "aa=4"
expect_exit "case3_count_zero" 0 "$VE" "$target" --count "zzz=0"

# Case 4: sed -i that matches nothing -> file unchanged, verify-edit exits 1
printf "keep this\\n" > "$target"
before=$(sha256sum "$target" | awk "{print \$1}")
sed -i "s/NEVER_MATCHES/xxx/" "$target"
after=$(sha256sum "$target" | awk "{print \$1}")
check "case4_sed_no_change" test "$before" = "$after"
expect_exit "case4_edit_not_landed" 1 "$VE" "$target" --contains xxx

# Case 5: missing/unreadable file exits 2; no args or unknown option exits 64
expect_exit "case5_missing_file" 2 "$VE" "$tmp/nope.txt" --contains x
printf "hi\\n" > "$target"
chmod 000 "$target"
if [ -r "$target" ]; then
    printf 'SKIP case5_unreadable (running as root)\n'
else
    expect_exit "case5_unreadable" 2 "$VE" "$target" --contains hi
fi
chmod 644 "$target"
expect_exit "case5_no_args" 64 "$VE"
expect_exit "case5_file_only" 64 "$VE" "$target"
expect_exit "case5_unknown_opt" 64 "$VE" "$target" --bogus x
expect_exit "case5_bad_count" 64 "$VE" "$target" --count "noequals"

# Case 6: special characters (regex metachars, slashes, quotes) matched literally
printf "a.b*c\npath/to/x\nquote'x\n" > "$target"
expect_exit "case6_dot_star" 0 "$VE" "$target" --contains "a.b*c"
expect_exit "case6_dot_literal_only" 1 "$VE" "$target" --contains "aXbXc"
expect_exit "case6_slash" 0 "$VE" "$target" --contains "path/to/x"
expect_exit "case6_quote" 0 "$VE" "$target" --contains "quote'x"

# Case 7: file never modified by verify-edit (checksum before/after)
printf "stable\\ncontent\\n" > "$target"
before=$(sha256sum "$target" | awk "{print \$1}")
"$VE" "$target" --contains stable >/dev/null 2>&1 || true
"$VE" "$target" --contains zzz >/dev/null 2>&1 || true
"$VE" "$target" --absent stable >/dev/null 2>&1 || true
after=$(sha256sum "$target" | awk "{print \$1}")
check "case7_readonly" test "$before" = "$after"

# Case 8: a directory (not a readable regular file) exits 2, never a false pass
mkdir -p "$tmp/adir"
expect_exit "case8_dir_absent" 2 "$VE" "$tmp/adir" --absent zzz
expect_exit "case8_dir_contains" 2 "$VE" "$tmp/adir" --contains zzz
expect_exit "case8_dir_count_zero" 2 "$VE" "$tmp/adir" --count "zzz=0"

# Case 9: no temp file is used, so an unwritable TMPDIR cannot turn a failed check into "landed"
printf "alpha\\nbeta\\n" > "$target"
expect_exit "case9_tmpdir_absent_needle" 1 env TMPDIR=/nonexistent "$VE" "$target" --contains ZZZ_NOT_THERE_ZZZ
expect_exit "case9_tmpdir_present_needle" 0 env TMPDIR=/nonexistent "$VE" "$target" --contains alpha

# Case 10: a multi-line TEXT must appear as one consecutive block
printf "x\\nQ\\ny\\n" > "$target"
expect_exit "case10_multiline_not_consecutive" 1 "$VE" "$target" --contains $'x\ny'
expect_exit "case10_multiline_absent_when_not_consecutive" 0 "$VE" "$target" --absent $'x\ny'
printf "x\\ny\\nz\\n" > "$target"
expect_exit "case10_multiline_consecutive" 0 "$VE" "$target" --contains $'x\ny'
expect_exit "case10_multiline_absent_when_present" 1 "$VE" "$target" --absent $'x\ny'
printf "a\\nb\\nx\\na\\nb\\n" > "$target"
expect_exit "case10_multiline_count_two" 0 "$VE" "$target" --count $'a\nb=2'
expect_exit "case10_multiline_count_wrong" 1 "$VE" "$target" --count $'a\nb=1'

# Case 11: --count counts occurrences (non-overlapping), not lines
printf "aaa\\n" > "$target"
expect_exit "case11_count_occurrences_three" 0 "$VE" "$target" --count "a=3"
expect_exit "case11_count_is_not_lines" 1 "$VE" "$target" --count "a=1"
expect_exit "case11_count_non_overlapping" 0 "$VE" "$target" --count "aa=1"
printf "a a a\\n" > "$target"
sed -i "s/a/b/g" "$target"
expect_exit "case11_overreplace_caught" 1 "$VE" "$target" --count "b=1"
expect_exit "case11_overreplace_exact" 0 "$VE" "$target" --count "b=3"

# Case 12: an empty TEXT is a usage error, alone or next to valid conditions
printf "alpha\\n" > "$target"
expect_exit "case12_empty_contains" 64 "$VE" "$target" --contains ""
expect_exit "case12_empty_absent" 64 "$VE" "$target" --absent ""
expect_exit "case12_empty_count_needle" 64 "$VE" "$target" --count "=3"
expect_exit "case12_empty_absent_after_valid" 64 "$VE" "$target" --contains alpha --absent ""
expect_exit "case12_empty_contains_before_valid" 64 "$VE" "$target" --contains "" --contains alpha

# Case 13: usage errors are reported before the file is looked at
expect_exit "case13_bad_count_missing_file" 64 "$VE" "$tmp/nope.txt" --count "x=abc"
expect_exit "case13_empty_needle_missing_file" 64 "$VE" "$tmp/nope.txt" --contains ""

# Case 14: several failed conditions are all reported on stderr; success prints nothing
printf "alpha\\nbeta\\n" > "$target"
err=$("$VE" "$target" --contains ZZZ --absent alpha --count "beta=5" 2>&1 >/dev/null); rc=$?
check "case14_combined_exit_1" test "$rc" -eq 1
check "case14_stderr_says_not_landed" bash -c 'printf "%s\n" "$1" | grep -q "not landed"' _ "$err"
check "case14_stderr_lists_every_reason" bash -c 'printf "%s\n" "$1" | grep -q "missing: ZZZ" && printf "%s\n" "$1" | grep -q "should be absent: alpha" && printf "%s\n" "$1" | grep -q "count for \[beta\]"' _ "$err"
out=$("$VE" "$target" --contains alpha 2>&1)
check "case14_success_is_silent" test -z "$out"

# Case 15: a file with a NUL byte cannot be verified faithfully -> exit 2
printf "ab\\000cd\\n" > "$target"
expect_exit "case15_nul_file" 2 "$VE" "$target" --contains ab

# Case 16: timing guard on a mid-size file: 20000 occurrences in a 10000-line file must
# finish well inside the timeout. It catches a copy-per-occurrence loop (about twice as
# slow here). It does not prove linearity: --count is quadratic in file size (see the header).
awk 'BEGIN { for (i = 1; i <= 10000; i++) print "line " i " token token" }' > "$tmp/big.txt"
expect_exit "case16_large_file_count" 0 timeout 8 "$VE" "$tmp/big.txt" --count "token=20000"
expect_exit "case16_large_file_count_wrong" 1 timeout 8 "$VE" "$tmp/big.txt" --count "token=19999"

# Case 17: TEXT with a slash, brackets, or glob characters is counted literally
printf "a/b a/b\\n[x] [x] [x]\\n" > "$target"
expect_exit "case17_slash_count" 0 "$VE" "$target" --count "a/b=2"
expect_exit "case17_bracket_count" 0 "$VE" "$target" --count "[x]=3"
expect_exit "case17_glob_is_not_pattern" 0 "$VE" "$target" --count "[a-z]=0"
expect_exit "case17_single_char_count" 0 "$VE" "$target" --count "x=3"

# Case 18: a wanted count too large for the shell's integer test is a usage error,
# never "landed" (fail closed)
printf "a\\na\\na\\n" > "$target"
expect_exit "case18_huge_count_is_usage_error" 64 "$VE" "$target" --count "a=99999999999999999999"
expect_exit "case18_two_pow_63_is_usage_error" 64 "$VE" "$target" --count "a=9223372036854775808"
expect_exit "case18_fifteen_digit_count_not_landed" 1 "$VE" "$target" --count "a=999999999999999"
expect_exit "case18_leading_zeros_same_value" 0 "$VE" "$target" --count "a=0000000000000000000003"
expect_exit "case18_leading_zeros_wrong_value" 1 "$VE" "$target" --count "a=0000000000000000000004"
expect_exit "case18_zero_written_with_zeros" 0 "$VE" "$target" --count "zzz=000000000000000000000"
err=$("$VE" "$target" --count "a=99999999999999999999" 2>&1 >/dev/null)
check "case18_no_shell_error_text" bash -c '! printf "%s\n" "$1" | grep -q "integer expression"' _ "$err"

# Case 19: TEXT is matched literally: glob characters, backslashes, a leading dash
printf "abc\\n" > "$target"
expect_exit "case19_contains_bracket_is_literal" 1 "$VE" "$target" --contains "[a-z]"
expect_exit "case19_contains_question_is_literal" 1 "$VE" "$target" --contains "a?c"
expect_exit "case19_contains_star_is_literal" 1 "$VE" "$target" --contains "a*c"
expect_exit "case19_absent_bracket_is_literal" 0 "$VE" "$target" --absent "[a-z]"
expect_exit "case19_absent_question_is_literal" 0 "$VE" "$target" --absent "a?c"
expect_exit "case19_absent_star_is_literal" 0 "$VE" "$target" --absent "a*c"
expect_exit "case19_count_question_is_literal" 0 "$VE" "$target" --count "?=0"
printf "x -n y\\n" > "$target"
expect_exit "case19_leading_dash_contains" 0 "$VE" "$target" --contains "-n"
expect_exit "case19_leading_dash_absent" 0 "$VE" "$target" --absent "-e"
printf "a\\\\b\\n" > "$target"
expect_exit "case19_backslash_contains" 0 "$VE" "$target" --contains 'a\b'
expect_exit "case19_backslash_not_doubled" 1 "$VE" "$target" --contains 'a\\b'
expect_exit "case19_backslash_absent_doubled" 0 "$VE" "$target" --absent 'a\\b'

# Case 20: UTF-8 content is compared byte for byte whatever the caller's locale
# (the script sets LC_ALL=C itself), and an empty file is a plain file with no text
printf "caf\\303\\251\\n" > "$target"
utf8env() { env LANG=C.UTF-8 LC_ALL= LC_CTYPE=C.UTF-8 "$@"; }
expect_exit "case20_utf8_contains_bytes" 0 utf8env "$VE" "$target" --contains "$(printf 'caf\303\251')"
expect_exit "case20_utf8_absent_other_text" 0 utf8env "$VE" "$target" --absent "cafe"
expect_exit "case20_utf8_not_landed_is_1_not_2" 1 utf8env "$VE" "$target" --contains x
expect_exit "case20_utf8_count_bytes" 0 utf8env "$VE" "$target" --count "$(printf '\303\251')=1"
: > "$target"
expect_exit "case20_empty_file_absent" 0 "$VE" "$target" --absent x
expect_exit "case20_empty_file_contains" 1 "$VE" "$target" --contains x
expect_exit "case20_empty_file_count_zero" 0 "$VE" "$target" --count "x=0"

# Case 21: the header states the known limit of --count
check "case21_quadratic_limit_documented" grep -q -F "quadratic" "$VE"
check "case21_digit_limit_documented" grep -q -F "15 digits" "$VE"

printf '\n%s passed, %s failed\n' "$pass" "$fail"
if [ "$fail" -gt 0 ]; then
    exit 1
fi
exit 0
