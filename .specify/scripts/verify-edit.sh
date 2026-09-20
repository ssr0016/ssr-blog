#!/usr/bin/env bash
# verify-edit.sh -- read-only edit verifier. Fixed-string matching.
# Usage: verify-edit.sh FILE [--contains TEXT] [--absent TEXT] [--count TEXT=N]
# Exit: 0 landed, 1 not landed, 2 could not verify, 64 bad usage.
set -u

TMP_LIST=$(mktemp)
trap 'rm -f "$TMP_LIST"' EXIT

usage() { printf "usage: verify-edit.sh FILE [--contains TEXT] [--absent TEXT] [--count TEXT=N]\n" >&2; exit 64; }

[ "$#" -ge 1 ] || usage
file="$1"; shift
[ -n "$file" ] || usage

contains_list=""
absent_list=""
count_list=""

append() {
    local var="$1"; local val="$2"
    if [ -z "${!var}" ]; then
        printf -v "$var" "%s" "$val"
    else
        printf -v "$var" "%s\n%s" "${!var}" "$val"
    fi
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --contains) [ "$#" -ge 2 ] || usage; append contains_list "$2"; shift 2 ;;
        --absent)   [ "$#" -ge 2 ] || usage; append absent_list "$2";   shift 2 ;;
        --count)
            [ "$#" -ge 2 ] || usage
            arg="$2"
            case "$arg" in
                *=*) ;;
                *) usage ;;
            esac
            append count_list "$arg"
            shift 2 ;;
        *) usage ;;
    esac
done

if [ -z "$contains_list$absent_list$count_list" ]; then
    usage
fi

if [ ! -e "$file" ] || [ ! -r "$file" ]; then
    printf "verify-edit: cannot read: %s\n" "$file" >&2
    exit 2
fi

landed=1
reasons=""
add_reason() {
    if [ -z "$reasons" ]; then
        reasons="$1"
    else
        reasons="$reasons; $1"
    fi
}

check_one_contains() {
    local needle="$1"
    [ -n "$needle" ] || return 0
    if ! grep -qF -- "$needle" "$file"; then
        add_reason "missing: $needle"
        landed=0
    fi
}

check_one_absent() {
    local needle="$1"
    [ -n "$needle" ] || return 0
    if grep -qF -- "$needle" "$file"; then
        add_reason "should be absent: $needle"
        landed=0
    fi
}

check_one_count() {
    local entry="$1"
    [ -n "$entry" ] || return 0
    local needle="${entry%=*}"
    local want="${entry##*=}"
    case "$want" in
        ""|*[!0-9]*) usage ;;
    esac
    local got
    got=$(grep -cF -- "$needle" "$file" || true)
    if [ "$got" -ne "$want" ]; then
        add_reason "count for [$needle]: want $want, got $got"
        landed=0
    fi
}

process_list() {
    local list="$1"; local fn="$2"
    [ -n "$list" ] || return 0
    local line
    printf "%s\n" "$list" > "$TMP_LIST"
    while IFS= read -r line; do
        "$fn" "$line"
    done < "$TMP_LIST"
}

process_list "$contains_list" check_one_contains
process_list "$absent_list" check_one_absent
process_list "$count_list" check_one_count

if [ "$landed" -eq 1 ]; then
    exit 0
fi

printf "verify-edit: not landed: %s\n" "$reasons" >&2
exit 1
