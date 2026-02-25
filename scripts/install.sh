#!/bin/sh
set -e

# --- Colors ---
RESET="\033[0m"
BOLD="\033[1m"
DIM="\033[2m"
GREEN="\033[38;2;74;222;128m"
DGREEN="\033[38;2;26;58;36m"

# Gradient: deep forest green → bright emerald (matches TUI shimmer palette)
# G       R       I       M       O       R       A
# Each letter gets a step from dark to bright and back
g1="\033[1;38;2;30;80;45m"    # G - deep
g2="\033[1;38;2;40;120;60m"   # R
g3="\033[1;38;2;52;160;82m"   # I
g4="\033[1;38;2;74;222;128m"  # M - peak emerald
g5="\033[1;38;2;52;160;82m"   # O
g6="\033[1;38;2;40;120;60m"   # R
g7="\033[1;38;2;30;80;45m"    # A - deep

logo() {
  printf "\n  ${g1}G ${g2}R ${g3}I ${g4}M ${g5}O ${g6}R ${g7}A${RESET}\n\n"
}

info() {
  printf "  ${DIM}%s${RESET}\n" "$1"
}

success() {
  printf "  ${GREEN}%s${RESET}\n" "$1"
}

fail() {
  printf "  ${BOLD}\033[31m%s${RESET}\n" "$1" >&2
  exit 1
}

# --- Platform detection ---
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) fail "Unsupported architecture: $ARCH" ;;
esac

case "$OS" in
  darwin|linux) ;;
  *) fail "Unsupported OS: $OS" ;;
esac

INSTALL_DIR="${HOME}/.local/bin"

logo
info "Platform: ${OS}/${ARCH}"
info "Install directory: ${INSTALL_DIR}"
echo ""

# --- Fetch latest version ---
info "Fetching latest release..."
VERSION=$(curl -fsSL https://api.github.com/repos/naveenspark/grimora/releases/latest | grep '"tag_name"' | sed 's/.*"v\([^"]*\)".*/\1/')
if [ -z "$VERSION" ]; then
  fail "Could not fetch latest release"
fi
info "Found version: v${VERSION}"

# --- Download + verify ---
TARBALL="grimora_${OS}_${ARCH}.tar.gz"
URL="https://github.com/naveenspark/grimora/releases/download/v${VERSION}/${TARBALL}"
CHECKSUMS_URL="https://github.com/naveenspark/grimora/releases/download/v${VERSION}/checksums.txt"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading ${TARBALL}..."
curl -fsSL -o "${TMPDIR}/${TARBALL}" "$URL"

# Verify checksum if checksums.txt exists
if curl -fsSL -o "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL" 2>/dev/null; then
  EXPECTED=$(grep "${TARBALL}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
  if [ -n "$EXPECTED" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL=$(sha256sum "${TMPDIR}/${TARBALL}" | awk '{print $1}')
    else
      ACTUAL=$(shasum -a 256 "${TMPDIR}/${TARBALL}" | awk '{print $1}')
    fi
    if [ "$EXPECTED" = "$ACTUAL" ]; then
      info "Checksum verified"
    else
      fail "Checksum mismatch — download may be corrupted"
    fi
  fi
fi

# --- Install ---
mkdir -p "$INSTALL_DIR"
tar xzf "${TMPDIR}/${TARBALL}" -C "$TMPDIR"
mv "${TMPDIR}/grimora" "${INSTALL_DIR}/grimora"
chmod +x "${INSTALL_DIR}/grimora"

echo ""
success "grimora v${VERSION} installed"
echo ""

# --- PATH setup ---
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    SHELL_NAME=$(basename "$SHELL")
    case "$SHELL_NAME" in
      zsh)
        # macOS Terminal.app opens login shells — .zprofile is more reliable than .zshrc
        if [ "$OS" = "darwin" ]; then
          RC_FILE="${HOME}/.zprofile"
        else
          RC_FILE="${HOME}/.zshrc"
        fi
        ;;
      bash)
        if [ "$OS" = "darwin" ]; then
          RC_FILE="${HOME}/.bash_profile"
        else
          RC_FILE="${HOME}/.bashrc"
        fi
        ;;
      *)    RC_FILE="" ;;
    esac

    if [ -n "$RC_FILE" ]; then
      if ! grep -q '\.local/bin' "$RC_FILE" 2>/dev/null; then
        printf '\nexport PATH="$HOME/.local/bin:$PATH"\n' >> "$RC_FILE"
        success "Added ~/.local/bin to PATH in ${RC_FILE##*/}"
      fi
      export PATH="${INSTALL_DIR}:${PATH}"
    else
      printf "  ${DIM}Add to your PATH:${RESET}\n"
      printf "  ${BOLD}  export PATH=\"\$HOME/.local/bin:\$PATH\"${RESET}\n"
      echo ""
    fi
    ;;
esac

# --- Install /grimora skill for AI coding tools ---
SKILL_CONTENT='Cast spells from Grimora'\''s library, or forge new ones for the Grimoire to judge.

## Modes
- `/grimora` — cast a spell from the library (default)
- `/grimora push` — forge a new spell and offer it to the Grimoire

---

Detect mode from `$ARGUMENTS`:
- If starts with "push" → Mode 2
- Otherwise → Mode 1

### Mode 1: Cast a Spell (default)

Call the Grimora API to find the right spell for the current situation.

**What to send:** A brief, GENERIC description of the technical situation.
Strip all project-specific details — no file paths, no project names,
no business context. Only the technical essence.

Good: "Debugging concurrency issues in a Go HTTP server"
Bad:  "Debugging the race condition in auth.go in our fintech app'\''s payment handler"

**Auth detection:** Check for `GRIMORA_TOKEN` to determine authenticated vs anonymous path.

```bash
source ~/.zshenv 2>/dev/null
if [ -n "$GRIMORA_TOKEN" ]; then
  curl -s -X POST https://grimora.ai/api/cast \
    -H '\''Content-Type: application/json'\'' \
    -H "Authorization: Bearer $GRIMORA_TOKEN" \
    -d '\''{"situation":"GENERIC TECHNICAL DESCRIPTION"}'\''
else
  curl -s -X POST https://grimora.ai/api/cast \
    -H '\''Content-Type: application/json'\'' \
    -d '\''{"situation":"GENERIC TECHNICAL DESCRIPTION"}'\''
fi
```

Present the result:

  ### Spell: {title — first H1 or context field}
  {author.login} · {author.guild_id} · {author.city || "unknown realm"} · Rank #{author.rank}
  {author.spells_forged} spells forged · {author.total_potency} potency · {upvotes} upvotes · {tag}

  {full spell text}

If author.rank is 0 (no forged spells), show "Unranked" instead of "Rank #0".

If `$GRIMORA_TOKEN` was NOT set (anonymous), append after the spell:

  ---
  Forge your own spells → grimora login

Then IMMEDIATELY apply the spell to the current task.

### Mode 2: Push (`/grimora push`)

Forge a spell from the current conversation and offer it to the Grimoire.

**Valid tags (MUST use one of these exactly):**
debugging, refactoring, architecture, testing, devops, data, frontend,
backend, security, performance, image-gen, writing, business, productivity,
analysis, system-prompt, education, coding, conversation, general

**Step 1: Extract and sanitize.**
Find the technique from the conversation. Write it as a GENERIC, transferable
spell — strip ALL project-specific details. One technique per spell.

**Step 2: Present for review.**

```
It is noble to share your craft with the library.
Forging a spell from your work...

───────────────────────────────────
Tag: {tag}

{full spell text, sanitized}
───────────────────────────────────

Review your spell. Edit anything, or say "submit" to offer it to the Grimoire.
```

WAIT for "submit". This is the only approval gate.

**Step 3: Submit.**

CRITICAL: Make exactly ONE Bash call. No token checks, no tag lookups, no
debug commands, no intermediate steps. ONE call:

```bash
source ~/.zshenv 2>/dev/null; curl -s -X POST "https://grimora.ai/api/forge" \
  -H '\''Content-Type: application/json'\'' \
  -H "Authorization: Bearer $GRIMORA_TOKEN" \
  -d '\''{"text":"SPELL_TEXT_JSON_ESCAPED","tag":"TAG"}'\''
```

**Step 4: Present the verdict.**

If ACCEPTED:
```
The Grimoire stirs.
"{inscription}"

Spell forged. Potency: {potency}/3 · {tag}

─── Your Craft ───
{stats.spells_forged} spells forged · {stats.acceptance_rate as %}% accepted · Rank #{stats.rank} of {stats.total_ranked}
Potency: {stats.total_potency} total ({stats.avg_potency:.1f} avg)
```

If REJECTED:
```
The Grimoire remains still.
"{reason}"

Spell not accepted. Refine and try again.

─── Your Craft ───
{stats.spells_forged} spells forged · {stats.acceptance_rate as %}% accepted · Rank #{stats.rank} of {stats.total_ranked}
Potency: {stats.total_potency} total ({stats.avg_potency:.1f} avg)
```

If stats.rank is 0 (no forged spells yet), show "Unranked" instead of "Rank #0".

If error: show the raw error. Don'\''t debug it.

$ARGUMENTS'

# Codex SKILL.md needs frontmatter
SKILL_FRONTMATTER='---
name: grimora
description: Cast spells from Grimora'\''s library, or forge new ones for the Grimoire to judge.
---
'

INSTALLED_SKILLS=""

# Claude Code: ~/.claude/commands/grimora.md
if [ -d "${HOME}/.claude" ] || command -v claude >/dev/null 2>&1; then
  mkdir -p "${HOME}/.claude/commands"
  printf '%s\n' "$SKILL_CONTENT" > "${HOME}/.claude/commands/grimora.md"
  INSTALLED_SKILLS="${INSTALLED_SKILLS} claude-code"
fi

# Codex: ~/.agents/skills/grimora/SKILL.md
if [ -d "${HOME}/.agents" ] || command -v codex >/dev/null 2>&1; then
  mkdir -p "${HOME}/.agents/skills/grimora"
  printf '%s\n%s\n' "$SKILL_FRONTMATTER" "$SKILL_CONTENT" > "${HOME}/.agents/skills/grimora/SKILL.md"
  INSTALLED_SKILLS="${INSTALLED_SKILLS} codex"
fi

# OpenCode: ~/.config/opencode/commands/grimora.md
if [ -d "${HOME}/.config/opencode" ] || command -v opencode >/dev/null 2>&1; then
  mkdir -p "${HOME}/.config/opencode/commands"
  printf '%s\n' "$SKILL_CONTENT" > "${HOME}/.config/opencode/commands/grimora.md"
  INSTALLED_SKILLS="${INSTALLED_SKILLS} opencode"
fi

if [ -n "$INSTALLED_SKILLS" ]; then
  success "/grimora skill installed for:${INSTALLED_SKILLS}"
else
  info "No AI coding tools detected (claude, codex, opencode)"
  info "Install one, then re-run this script to get /grimora"
fi
echo ""

# --- Get started ---
printf "  Get started:\n"
printf "    ${BOLD}grimora login${RESET}       ${DIM}# authenticate with GitHub${RESET}\n"
printf "    ${BOLD}grimora${RESET}             ${DIM}# enter the Hall${RESET}\n"
printf "    ${BOLD}grimora update${RESET}      ${DIM}# check for updates${RESET}\n"
echo ""
