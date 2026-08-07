# Notes-CLI

A command-line interface tool written in Rust designed to create and manage Markdown notes. It seamlessly integrates with **Obsidian** vaults by outputting standard `.md` files complete with tags and optional AI-powered summaries powered by the Groq API.

---

## Features

* **Obsidian Integration:** Creates Markdown files with internal link tag formatting — each tag becomes its own wikilink (`tags: [[tag1]], [[tag2]]`).


* **AI Summarization:** Uses Groq API (`llama-3.1-8b-instant`) to auto-summarize input text or transcripts.


* **Configurable Paths:** Save your default Obsidian vault directory to avoid typing paths repeatedly.


* **Interactive Input:** Accept text arguments directly or paste long transcripts straight from `stdin`.


* **Self-Cleaner:** Built-in `uninstall` command to remove user config directories and binary files.

* **Cross-Platform File Handling:** Note filenames are built with the OS-native path separator and sanitized against characters that aren't valid in filenames (`/ \ : * ? " < > |`), so notes save correctly on Windows, macOS, and Linux even if the title contains unusual characters.



---

## Global Options & Flags

| Flag / Option | Description |
| --- | --- |
| `-l`, `--list` | Lists all `.md` files in the target folder after executing an action.|
| `-h`, `--help` | Prints help information. |
| `-V`, `--version` | Prints version information.|

---

## Commands & Usage Examples

### 1. Set Default Output Directory

Set and persist the default output path (e.g., your Obsidian Vault location).

```bash
notes-cli set-path <PATH>

```

**Example:**

```bash
notes-cli set-path "D:/ObsidianVault/Notes"

```

---

### 2. Set Groq API Key

Explicitly save your Groq API key without going through the interactive prompt. This is the way to go if you're running `notes-cli` somewhere without a real terminal to prompt in (a script, CI, an IDE "run" button, `docker exec` without `-it`, etc.) — in those environments the interactive prompt used by `add -s` can't ask you anything and will just fail.

```bash
notes-cli set-key <KEY>

```

**Example:**

```bash
notes-cli set-key gsk_abc123yourkeyhere

```

This saves the key to the same OS Keychain/Credential Manager location the interactive prompt uses, so once it's set this way you won't be prompted again. You can also skip persistence entirely and just set the `GROQ_API_KEY` environment variable for the current shell/session — see [API Key Storage & Error Handling](#api-key-storage--error-handling) below for the full lookup order.

> **Note:** the key you pass to `set-key` will be visible in your shell history and in process listings (e.g. `ps`) while the command runs. If that's a concern on a shared machine, prefer the interactive `add -s` prompt or the `GROQ_API_KEY` environment variable instead.

---

### 3. Clear Groq API Key

Remove the stored Groq API key from the OS keychain without replacing it.

```bash
notes-cli clear-key

```

You normally won't need this — `notes-cli` automatically detects and clears an empty or corrupted stored key (e.g. one containing stray control characters) and re-prompts you in the same run — but it's there as a manual reset if you ever want one.

---

### 4. Add a Note

Creates a new `.md` note in the configured destination path or a specified custom directory.

```bash
notes-cli add <TITLE> [BODY] [FLAGS]

```

#### Flags for `add`

| Flag / Option | Type | Description |
| --- | --- | --- |
| `-t`, `--tags` | List | Comma-separated list of tags (up to 5). Each tag is wrapped in its own Obsidian wikilink: `tags: [[tag1]], [[tag2]]` — not one link around the whole list. Whitespace around each tag is trimmed automatically, so `-t "rust, coding, notes"` and `-t rust,coding,notes` produce the same clean tags, and any empty entries from stray/double commas are dropped.|
| `-p`, `--path` | String | Override the default output folder for this note only.|
| `-s`, `--summarize` | Flag | Calls Groq AI to generate a summary at the top of the note.|

---

#### Use Case Examples

**Basic Note with Inline Text:**

```bash
notes-cli add "Meeting Notes" "Discussed Q3 roadmaps and budget allocations."

```

**Note with Tags:**

```bash
notes-cli add "Rust Memory Model" "Ownership and borrowing are key concepts." -t rust,coding,notes

```

**Note with Spaced Tags:**
Tags are trimmed automatically, so it doesn't matter whether you leave spaces after the commas.

```bash
notes-cli add "Rust Memory Model" "Ownership and borrowing are key concepts." -t "rust, coding, notes"

```

**Note with Custom Folder Path & File Listing:**

```bash
notes-cli add "Project Alpha" "Initial brainstorming details." -p "C:/Vault/Work" --list

```

**Interactive Multi-line Entry / Transcript via `stdin`:**
If you omit the `BODY` argument, `notes-cli` prompts you to paste input. Press `Ctrl+D` (Linux/macOS) or `Ctrl+Z` then `Enter` (Windows) to complete.

```bash
notes-cli add "Lecture Transcript" --tags lecture,university

```

**AI Summarization (`--summarize` / `-s`):**
Summarizes input using the Groq API (`llama-3.1-8b-instant`). If no API key is stored, you will be prompted to enter your Groq API key on first use.

```bash
notes-cli add "Podcast Highlights" "Today we discussed artificial intelligence..." -s -t ai,podcast

```

*Generated Markdown format when using `-s`:*

```markdown
# Podcast Highlights

tags: [[ai]], [[podcast]]

## Summary
- Key takeaway 1...
- Key takeaway 2...

---

## Transcript
Today we discussed artificial intelligence...

```

---

### 5. Uninstall

Removes user configuration files (located at `~/.config/notes-cli` or system equivalent) and removes the installed executable binary.

```bash
notes-cli uninstall

```

**Example Output:**

```bash
Are you sure you want to uninstall Notes-CLI? (y/N): y
Cleared notes-cli configuration directory.
Removing executable at: /usr/local/bin/notes-cli
Notes-CLI has been successfully uninstalled.

```

---

## Configuration File

Configurations are stored automatically in JSON format under the OS standard configuration directory (e.g., `%APPDATA%\notes-cli\config.json` on Windows or `~/.config/notes-cli/config.json` on Unix).

```json
{
  "destination_path": "D:/ObsidianVault/Notes"
}

```

Your Groq API key is **not** stored in this file — it lives in your OS's native credential store (macOS Keychain, Windows Credential Manager, or the Linux Secret Service), which is more secure than a plaintext JSON file. See [API Key Storage & Error Handling](#api-key-storage--error-handling) below for exactly how and where it's kept.

## How to Get a Groq API KeyTo use the AI

To use the AI summarization feature (-s or --summarize), you will need a free API key from Groq:  

1. Sign Up: Go to console.groq.com and create a free account.
2. Navigate to API Keys: Once logged in, select API Keys from the sidebar menu.
3. Create Key: Click Create API Key, give it a name (e.g., notes-cli), and copy the generated key string (it typically starts with gsk_).
4. Usage in Notes-CLI: Either run `notes-cli set-key <KEY>` directly, or just run an `add` command with `--summarize`/`-s` — if no key is found, you'll be prompted to paste one interactively. Either way, it's saved for future runs.

---

## API Key Storage & Error Handling

`notes-cli` looks for your Groq API key in this order:

1. The `GROQ_API_KEY` environment variable, if set and non-empty.
2. The OS keychain / credential manager (macOS Keychain, Windows Credential Manager, or the Linux Secret Service), if a key was saved previously — either via `notes-cli set-key <KEY>` or a prior interactive prompt.
3. An interactive prompt, if none of the above has a key. **This requires a real terminal (TTY).** If you're running `notes-cli` from a script, CI, or anywhere else without one, this step will fail — use `notes-cli set-key <KEY>` or the `GROQ_API_KEY` environment variable instead.

**Entering a key:**
* Via `notes-cli set-key <KEY>`: saved immediately without being tested against the API — this is a deliberate, explicit action, so it's trusted as-is. Useful for non-interactive environments.
* Via the interactive prompt (triggered automatically by `add -s` when no key is found): if you press Enter without typing anything, you'll be re-prompted — up to 3 attempts — instead of the command failing outright. **A key entered this way is only written to the OS keychain after Groq actually accepts it** — you won't see a "saved" message, and nothing gets persisted, for a key that turns out to be invalid.
* Keys are expected to start with `gsk_`. If yours doesn't, `notes-cli` warns you but still lets you continue (in case Groq's key format changes).
* If the OS keychain isn't available (e.g. some headless Linux/CI environments), `notes-cli` warns you and continues without persisting the key — you'll just be asked again on your next run.

**If Groq rejects the key:**
* A `401 Unauthorized` response means the key is invalid or expired. If it came from the keychain, `notes-cli` clears it; either way it immediately re-prompts you for a new one **in the same run** (up to 2 attempts total) — you won't need to re-run the command, and nothing invalid stays cached.
* A corrupted or malformed key (e.g. containing stray control characters, which can otherwise cause a confusing "failed to parse header value" error) is detected and rejected the same way, before it ever reaches the network.
* A `429` response means you've hit Groq's rate limit; wait a moment and try again.
* Other API errors are shown with Groq's own error message where available.

**Network issues:**
* Requests to Groq time out after 30 seconds rather than hanging indefinitely.
* Connection failures and timeouts are reported with a specific message so you know whether it's a connectivity issue rather than a bad key.

**Note:** if `--summarize` fails for any reason, `notes-cli` still creates your note — it just skips the AI summary and prints the error to the console, so you never lose a transcript or note body because of an API problem.