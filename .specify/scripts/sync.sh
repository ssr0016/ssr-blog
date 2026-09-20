#!/usr/bin/env bash
# sync.sh -- compare and install workflow helpers between the repo and the home copy.
# The repo is the source of truth. SPEC_KIT_HOME (default ~/.spec-kit) holds installed copies.
# Neither side is ever changed silently: check is read-only, and install refuses to
# overwrite a differing home file unless --force is given (the old file is backed up).
# Usage:
#   sync.sh check
#   sync.sh install [--force]
# Exit: 0 all identical or installed, 1 drift or refused, 2 unreadable or setup error, 64 usage.
set -u

fail() { local code="$1"; shift; printf 'sync: %s\n' "$*" >&2; exit "$code"; }

usage() {
    printf 'usage: sync.sh check | sync.sh install [--force]\n' >&2
    exit 64
}

mode="${1:-}"
[ -n "$mode" ] || usage
shift
force=0
case "$mode" in
    check) [ "$#" -eq 0 ] || usage ;;
    install)
        while [ "$#" -gt 0 ]; do
            case "$1" in
                --force) force=1; shift ;;
                *) usage ;;
            esac
        done ;;
    *) usage ;;
esac

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
home="${SPEC_KIT_HOME:-}"
if [ -z "$home" ]; then
    [ -n "${HOME:-}" ] || fail 2 "neither SPEC_KIT_HOME nor HOME is set"
    home="$HOME/.spec-kit"
fi

# Repo-relative sources. Each is installed flat as $home/<basename>.
SRCS=".specify/scripts/writefile.sh .specify/scripts/verify-edit.sh .specify/scripts/sync.sh .specify/converge.yaml"

for rel in $SRCS; do
    [ -r "$REPO/$rel" ] || fail 2 "repo source not readable: $REPO/$rel (run from a repo checkout)"
done

sha() { sha256sum -- "$1" | awk '{print $1}'; }

# classify REL -> missing | unreadable | identical | differ
classify() {
    local src="$REPO/$1"
    local dst="$home/$(basename "$1")"
    if [ ! -e "$dst" ]; then
        echo missing
    elif [ ! -f "$dst" ] || [ ! -r "$dst" ]; then
        echo unreadable
    elif [ "$(sha "$src")" = "$(sha "$dst")" ]; then
        echo identical
    else
        echo differ
    fi
}

n_total=0
n_identical=0
n_missing=0
n_differ=0
n_unreadable=0

# scan: print one status line per file and set the counters.
scan() {
    local rel st
    for rel in $SRCS; do
        st="$(classify "$rel")"
        printf '%-10s %s\n' "$st" "$(basename "$rel")"
        n_total=$((n_total + 1))
        case "$st" in
            identical)  n_identical=$((n_identical + 1)) ;;
            missing)    n_missing=$((n_missing + 1)) ;;
            differ)     n_differ=$((n_differ + 1)) ;;
            unreadable) n_unreadable=$((n_unreadable + 1)) ;;
        esac
    done
}

if [ "$mode" = "check" ]; then
    scan
    if [ "$n_unreadable" -gt 0 ]; then
        printf 'check: %s of %s not readable\n' "$n_unreadable" "$n_total"
        exit 2
    fi
    if [ "$n_missing" -gt 0 ] || [ "$n_differ" -gt 0 ]; then
        printf 'check: %s of %s need attention (%s missing, %s differing)\n' \
            "$((n_missing + n_differ))" "$n_total" "$n_missing" "$n_differ"
        exit 1
    fi
    printf 'ok: %s of %s identical\n' "$n_identical" "$n_total"
    exit 0
fi

# install
scan
if [ "$n_unreadable" -gt 0 ]; then
    fail 2 "$n_unreadable home file(s) not readable; nothing installed"
fi
if [ "$n_differ" -gt 0 ] && [ "$force" -eq 0 ]; then
    fail 1 "refusing to overwrite $n_differ differing file(s); nothing installed (use --force to overwrite with backup)"
fi

mkdir -p -- "$home" || fail 2 "cannot create $home"

todo_rel=()
todo_tmp=()
cleanup() {
    local t
    if [ "${#todo_tmp[@]}" -gt 0 ]; then
        for t in "${todo_tmp[@]}"; do rm -f -- "$t"; done
    fi
}
trap cleanup EXIT

# Stage every copy first, so a failure leaves the home directory unchanged.
for rel in $SRCS; do
    st="$(classify "$rel")"
    case "$st" in
        missing|differ) ;;
        *) continue ;;
    esac
    tmp="$(mktemp "$home/.sync.XXXXXX")" || fail 2 "cannot stage in $home"
    todo_rel+=("$rel")
    todo_tmp+=("$tmp")
    cp -- "$REPO/$rel" "$tmp" && chmod --reference="$REPO/$rel" "$tmp" || fail 2 "cannot stage $rel"
done

ts="$(date -u +%Y-%m-%dT%H-%M-%S)"
n=${#todo_rel[@]}
n_installed=0
n_updated=0

# Back up every differing file before any file is replaced.
i=0
while [ "$i" -lt "$n" ]; do
    dst="$home/$(basename "${todo_rel[$i]}")"
    if [ -e "$dst" ]; then
        cp -p -- "$dst" "$dst.bak.$ts" || fail 2 "cannot back up $dst"
    fi
    i=$((i + 1))
done

i=0
while [ "$i" -lt "$n" ]; do
    rel="${todo_rel[$i]}"
    dst="$home/$(basename "$rel")"
    if [ -e "$dst" ]; then
        mv -- "${todo_tmp[$i]}" "$dst" || fail 2 "cannot replace $dst"
        printf 'updated    %s (backup %s.bak.%s)\n' "$(basename "$rel")" "$(basename "$rel")" "$ts"
        n_updated=$((n_updated + 1))
    else
        mv -- "${todo_tmp[$i]}" "$dst" || fail 2 "cannot install $dst"
        printf 'installed  %s\n' "$(basename "$rel")"
        n_installed=$((n_installed + 1))
    fi
    i=$((i + 1))
done

printf 'install: %s installed, %s updated, %s unchanged\n' "$n_installed" "$n_updated" "$n_identical"
exit 0
