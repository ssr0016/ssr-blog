#!/usr/bin/env bash
# writefile.sh -- paste-safe file writer with verified, backed-up writes.
# Usage:
#   writefile.sh TARGET --lines N --sig TEXT [--sha256 HEX] [--no-backup] [--allow-utf8] < content
#   writefile.sh TARGET --from FILE [--no-backup] [--allow-utf8]
# With content on stdin, --lines and --sig are both required: the sender states what to
# expect independently of the payload, or the check would only verify this script's own
# write. TARGET must not be a directory.
# After every successful write, files in the target's directory named
# *.bak.YYYY-MM-DDTHH-MM-SS (dotfile backups included) whose timestamp is over 7 days old
# are deleted, whether or not this script made them (accepted risk). Other names are never
# touched, and a cleanup failure never fails a write that already happened.
set -eu

die() { printf "writefile: %s\\n" "$*" >&2; exit 1; }

[ "$#" -ge 1 ] || die "usage: writefile.sh TARGET [options] < content"
target="$1"; shift
[ -n "$target" ] || die "TARGET must not be empty"

lines_req=""
sig_req=""
sha_req=""
from_file=""
no_backup=0
allow_utf8=0

while [ "$#" -gt 0 ]; do
    case "$1" in
        --lines) [ "$#" -ge 2 ] || die "--lines needs a value"; lines_req="$2"; shift 2 ;;
        --sig)   [ "$#" -ge 2 ] || die "--sig needs a value";   sig_req="$2";   shift 2 ;;
        --sha256) [ "$#" -ge 2 ] || die "--sha256 needs a value"; sha_req="$2"; shift 2 ;;
        --from)  [ "$#" -ge 2 ] || die "--from needs a value";  from_file="$2"; shift 2 ;;
        --no-backup)   no_backup=1; shift ;;
        --allow-utf8)  allow_utf8=1; shift ;;
        *) die "unknown option: $1" ;;
    esac
done

if [ -n "$from_file" ]; then
    [ -z "$lines_req$sig_req$sha_req" ] || die "--from cannot be combined with --lines/--sig/--sha256"
else
    [ -n "$lines_req" ] || die "--lines is required when content comes from stdin"
    [ -n "$sig_req" ] || die "--sig is required when content comes from stdin"
fi

[ ! -d "$target" ] || die "TARGET is a directory: $target"

tmpdir="$(dirname "$target")"
[ -d "$tmpdir" ] || die "target directory does not exist: $tmpdir"

tmp="$(mktemp "$tmpdir/.writefile.XXXXXX")"
cleanup_tmp() { rm -f "$tmp"; }
trap cleanup_tmp EXIT

if [ -n "$from_file" ]; then
    [ -f "$from_file" ] || die "--from file not found: $from_file"
    [ -n "$lines_req" ] || lines_req=$(wc -l < "$from_file")
    [ -n "$sig_req" ]   || sig_req=$(tail -n1 "$from_file")
    cp -- "$from_file" "$tmp"
else
    cat > "$tmp"
fi

if [ ! -s "$tmp" ]; then
    die "refusing to write empty content"
fi

if [ "$allow_utf8" -eq 0 ]; then
    bad=$(LC_ALL=C awk '{
        for (i = 1; i <= length($0); i++) {
            c = substr($0, i, 1)
            if (c < " " && c != "\t") { printf "%d:%d\n", NR, i; exit }
            if (c > "~") { printf "%d:%d\n", NR, i; exit }
        }
    }' "$tmp" || true)
    if [ -n "$bad" ]; then
        die "non-ASCII byte at line:column $bad"
    fi
fi

if [ -n "$lines_req" ]; then
    actual=$(wc -l < "$tmp")
    [ "$actual" -eq "$lines_req" ] || die "line count mismatch: expected $lines_req, got $actual"
fi

if [ -n "$sig_req" ]; then
    if ! tail -n1 "$tmp" | grep -qFx -- "$sig_req"; then
        die "signature mismatch: last line is not: $sig_req"
    fi
fi

if [ -n "$sha_req" ]; then
    actual_sha=$(sha256sum "$tmp" | awk "{print \$1}")
    [ "$actual_sha" = "$sha_req" ] || die "sha256 mismatch: expected $sha_req, got $actual_sha"
fi

if [ -e "$target" ] && [ "$no_backup" -eq 0 ]; then
    ts=$(date -u +%Y-%m-%dT%H-%M-%S)
    backup="$target.bak.$ts"
    if ! cp -- "$target" "$backup"; then
        die "failed to create backup: $backup"
    fi
fi

mv -- "$tmp" "$target"
trap - EXIT

# Best effort: a cleanup problem must never fail or undo a write that already happened.
prune_old_backups() {
    local cutoff old old_ts
    cutoff=$(date -u -d "7 days ago" +%Y-%m-%dT%H-%M-%S 2>/dev/null) || return 0
    # Two globs: the first does not match names that start with a dot, so the second
    # covers the backups of dotfile targets such as .gitignore.
    for old in "$tmpdir"/*.bak.* "$tmpdir"/.*.bak.*; do
        [ -f "$old" ] || continue
        old_ts=${old##*.bak.}
        case "$old_ts" in
            [0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]-[0-9][0-9]-[0-9][0-9])
                if [ "$old_ts" \< "$cutoff" ]; then rm -f -- "$old" 2>/dev/null || true; fi ;;
        esac
    done
    return 0
}
prune_old_backups
exit 0
