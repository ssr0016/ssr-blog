#!/usr/bin/env bash
# setup-home.sh
# Setup home-level AI workflow files after cloning the repo

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=========================================="
echo " Setup Home-Level AI Workflow"
echo " Repo: $REPO_ROOT"
echo "=========================================="
echo ""

# 1. CLAUDE COMMANDS
echo "[1/3] Claude commands..."
mkdir -p "$HOME/.claude/commands"

if [ -d "$REPO_ROOT/.claude/commands" ]; then
  cp "$REPO_ROOT/.claude/commands/"*.md "$HOME/.claude/commands/" 2>/dev/null || true
  COUNT=$(ls "$REPO_ROOT/.claude/commands/"*.md 2>/dev/null | wc -l)
  echo "      Copied $COUNT commands to ~/.claude/commands/"
else
  echo "      SKIP (no .claude/commands/ in repo)"
fi
echo ""

# 2. DEEPSEEK WRAPPER
echo "[2/3] DeepSeek wrapper..."
if [ -f "$HOME/bin/ds" ]; then
  echo "      SKIP (already exists)"
else
  mkdir -p "$HOME/bin"
  echo "      NOTE: ~/bin/ds not found. Set up manually if needed."
fi
echo ""

# 3. PATH
echo "[3/3] PATH..."
if grep -q 'HOME/bin' "$HOME/.bashrc" 2>/dev/null; then
  echo "      SKIP (PATH already set)"
else
  echo 'export PATH="$HOME/bin:$PATH"' >> "$HOME/.bashrc"
  echo "      Added ~/bin to PATH"
fi
echo ""

echo "=========================================="
echo " DONE"
echo "=========================================="
echo ""
echo "Next steps:"
echo "  1. source ~/.bashrc"
echo "  2. claude    (start Claude Code)"
echo "  3. Optional: create ~/.ai-workflow/global/AGENTS.md"
echo ""
