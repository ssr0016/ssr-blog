#!/usr/bin/env bash
# test_no_heredoc.sh -- bans multi-line heredoc markers in shipped scripts
# and inside fenced code blocks of command/prompt markdown.
#
# Rule source: T6 workflow-change rules and spec behavior [15-20].
# Scripts: every top-level *.sh under .specify/scripts/ must contain no
#          heredoc marker token. Test files under .specify/scripts/test/
#          are exempt: they embed the token as fixture data on purpose.
# Markdown: a heredoc marker inside a fenced code block is banned; prose
#           mentions outside a fence are allowed.
#
# Self-contained. No bats, no shellcheck.

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

PASS=0
FAIL=0

ok()   { PASS=$((PASS+1)); printf 'ok   - %s\n' "$1"; }
bad()  { FAIL=$((FAIL+1)); printf 'FAIL - %s\n' "$1"; }

# Marker token, built at runtime so this file never contains it literally.
MARK="$(printf '<%s' '<')"

# --- scanner: top-level scripts under .specify/scripts/ ---
scan_scripts() {
  local f hits=0
  while IFS= read -r f; do
    if grep -qF "$MARK" "$f"; then
      bad "heredoc marker in script: ${f#$REPO_ROOT/}"
      grep -nF "$MARK" "$f" | sed "s/^/       /"
      hits=$((hits+1))
    else
      ok "no heredoc marker: ${f#$REPO_ROOT/}"
    fi
  done < <(find "$REPO_ROOT/.specify/scripts" -maxdepth 1 -name "*.sh" -type f | sort)
  return 0
}

# --- scanner: markdown command/prompt files, fences only ---
scan_md_fences() {
  local f line infence=0 lineno=0 hitfile=0
  FENCE="$(printf '```')"
  while IFS= read -r f; do
    infence=0; lineno=0; hitfile=0
    while IFS= read -r line; do
      lineno=$((lineno+1))
      case "$line" in
          "$FENCE"*)
          if [ "$infence" -eq 0 ]; then infence=1; else infence=0; fi
          continue ;;
      esac
      if [ "$infence" -eq 1 ] && printf "%s" "$line" | grep -qF "$MARK"; then
        bad "heredoc marker in fence: ${f#$REPO_ROOT/}:$lineno"
        hitfile=$((hitfile+1))
      fi
    done < "$f"
    if [ "$hitfile" -eq 0 ]; then
      ok "no heredoc marker in fences: ${f#$REPO_ROOT/}"
    fi
  done < <(find "$REPO_ROOT/.claude/commands" "$REPO_ROOT/.ai-workflow/prompts" -maxdepth 1 -name "*.md" -type f 2>/dev/null | sort)
  return 0
}

main() {
  scan_scripts
  scan_md_fences
  printf "\n%d passed, %d failed\n" "$PASS" "$FAIL"
  [ "$FAIL" -eq 0 ]
}

main "$@"
