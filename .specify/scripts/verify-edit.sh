#!/usr/bin/env bash
# verify-edit.sh -- read-only edit verifier. Literal (fixed-string) matching.
# Usage: verify-edit.sh FILE [--contains TEXT] [--absent TEXT] [--count TEXT=N]
#   TEXT must not be empty. TEXT may span several lines: it then has to appear as
#   one consecutive block. --count counts non-overlapping occurrences of TEXT,
#   not lines. FILE must be a readable regular file without NUL bytes.
#   A wanted count of more than 15 digits (after leading zeros) is a usage error: it does
#   not fit the shell's integer test, and a silent false pass is worse than a refusal.
#   Limit: --count is quadratic in file size (about 60 ms for 200 KB, 4 s for 2 MB, minutes
#   for 10 MB). That is fine for workflow files (tens of KB); do not use it on huge files.
#   --contains and --absent stay fast on large files.
# Exit: 0 landed, 1 not landed, 2 could not verify, 64 bad usage.
# The file is read once into memory and matched with bash itself: no temp file,
# no grep, so no tool failure can be mistaken for "no match".
set -u
export LC_ALL=C

usage() { printf "usage: verify-edit.sh FILE [--contains TEXT] [--absent TEXT] [--count TEXT=N]\n" >&2; exit 64; }

[ "$#" -ge 1 ] || usage
file="$1"; shift
[ -n "$file" ] || usage

contains_list=()
absent_list=()
count_needles=()
count_wants=()

while [ "$#" -gt 0 ]; do
    case "$1" in
        --contains)
            [ "$#" -ge 2 ] || usage
            [ -n "$2" ] || usage
            contains_list+=("$2")
            shift 2 ;;
        --absent)
            [ "$#" -ge 2 ] || usage
            [ -n "$2" ] || usage
            absent_list+=("$2")
            shift 2 ;;
        --count)
            [ "$#" -ge 2 ] || usage
            arg="$2"
            case "$arg" in
                *=*) ;;
                *) usage ;;
            esac
            needle="${arg%=*}"
            want="${arg##*=}"
            [ -n "$needle" ] || usage
            case "$want" in
                ""|*[!0-9]*) usage ;;
            esac
            # Drop leading zeros, then refuse what the shell's integer test cannot hold.
            while [ "${#want}" -gt 1 ] && [ "${want:0:1}" = "0" ]; do want="${want:1}"; done
            [ "${#want}" -le 15 ] || usage
            count_needles+=("$needle")
            count_wants+=("$want")
            shift 2 ;;
        *) usage ;;
    esac
done

n_contains=${#contains_list[@]}
n_absent=${#absent_list[@]}
n_count=${#count_needles[@]}

if [ "$((n_contains + n_absent + n_count))" -eq 0 ]; then
    usage
fi

if [ ! -f "$file" ] || [ ! -r "$file" ]; then
    printf "verify-edit: cannot read as a regular file: %s\n" "$file" >&2
    exit 2
fi

# Read the whole file. The trailing x keeps final newlines; a NUL byte would be
# dropped by the shell, which shows up as a length mismatch, so it is refused.
size=$(wc -c < "$file") || { printf "verify-edit: cannot read: %s\n" "$file" >&2; exit 2; }
if ! { content=$(cat -- "$file" && printf x); } 2>/dev/null; then
    printf "verify-edit: cannot read: %s\n" "$file" >&2
    exit 2
fi
content="${content%x}"
if [ "${#content}" -ne "$size" ]; then
    printf "verify-edit: cannot verify (NUL byte or file changed while reading): %s\n" "$file" >&2
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
    landed=0
}

# count_occurrences TEXT -> number of non-overlapping occurrences in $content.
# One pass: delete every occurrence, then divide the removed length by the length of TEXT.
count_occurrences() {
    local text="$1"
    local stripped="${content//"$text"/}"
    printf '%s\n' "$(( (${#content} - ${#stripped}) / ${#text} ))"
}

i=0
while [ "$i" -lt "$n_contains" ]; do
    text="${contains_list[$i]}"
    if [[ "$content" != *"$text"* ]]; then
        add_reason "missing: $text"
    fi
    i=$((i + 1))
done

i=0
while [ "$i" -lt "$n_absent" ]; do
    text="${absent_list[$i]}"
    if [[ "$content" == *"$text"* ]]; then
        add_reason "should be absent: $text"
    fi
    i=$((i + 1))
done

i=0
while [ "$i" -lt "$n_count" ]; do
    text="${count_needles[$i]}"
    want="${count_wants[$i]}"
    got="$(count_occurrences "$text")"
    # "! -eq" rather than "-ne": if the test itself ever errors, that counts as not landed.
    if ! [ "$got" -eq "$want" ]; then
        add_reason "count for [$text]: want $want, got $got"
    fi
    i=$((i + 1))
done

if [ "$landed" -eq 1 ]; then
    exit 0
fi

printf "verify-edit: not landed: %s\n" "$reasons" >&2
exit 1
