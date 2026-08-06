# Notes-CLI

A command-line interface tool written in Rust designed to create and manage Markdown notes. It seamlessly integrates with **Obsidian** vaults by outputting standard `.md` files complete with tags and optional AI-powered summaries powered by the Groq API.

---

## Features

* **Obsidian Integration:** Creates Markdown files with internal link tag formatting (`tags: [[tag1, tag2]]`).


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

### 2. Add a Note

Creates a new `.md` note in the configured destination path or a specified custom directory.

```bash
notes-cli add <TITLE> [BODY] [FLAGS]

```

#### Flags for `add`

| Flag / Option | Type | Description |
| --- | --- | --- |
| `-t`, `--tags` | List | Comma-separated list of tags (up to 5). Formatted as `tags: [[tag1, tag2]]`.|
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

tags: [[ai, podcast]]

## Summary
- Key takeaway 1...
- Key takeaway 2...

---

## Transcript
Today we discussed artificial intelligence...

```

---

### 3. Uninstall

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
  "destination_path": "D:/ObsidianVault/Notes",
  "groq_api_key": "gsk_..."
}

```
## How to Get a Groq API KeyTo use the AI

To use the AI summarization feature (-s or --summarize), you will need a free API key from Groq:  

1. Sign Up: Go to console.groq.com and create a free account.
2. Navigate to API Keys: Once logged in, select API Keys from the sidebar menu.
3. Create Key: Click Create API Key, give it a name (e.g., notes-cli), and copy the generated key string (it typically starts with gsk_).
4. Usage in Notes-CLI: The first time you run an add command with the --summarize or -s flag, notes-cli will prompt you to paste your API key. Once entered, it will save it locally to your configuration for future uses.

---

## API Key Storage & Error Handling

`notes-cli` looks for your Groq API key in this order:

1. The `GROQ_API_KEY` environment variable, if set and non-empty.
2. The OS keychain / credential manager (macOS Keychain, Windows Credential Manager, or the Linux Secret Service), if a key was saved on a previous run.
3. An interactive prompt, if neither of the above has a key.

**Entering a key:**
* If you press Enter without typing anything, you'll be re-prompted — up to 3 attempts — instead of the command failing outright.
* Keys are expected to start with `gsk_`. If yours doesn't, `notes-cli` warns you but still lets you continue (in case Groq's key format changes).
* If the OS keychain isn't available (e.g. some headless Linux/CI environments), `notes-cli` warns you and continues without persisting the key — you'll just be asked again on your next run.

**If Groq rejects the key:**
* A `401 Unauthorized` response means the stored key is invalid or expired. `notes-cli` automatically clears it from the keychain and tells you to re-run the command so you can enter a fresh one — you won't get stuck retrying a bad cached key forever.
* A `429` response means you've hit Groq's rate limit; wait a moment and try again.
* Other API errors are shown with Groq's own error message where available.

**Network issues:**
* Requests to Groq time out after 30 seconds rather than hanging indefinitely.
* Connection failures and timeouts are reported with a specific message so you know whether it's a connectivity issue rather than a bad key.

**Note:** if `--summarize` fails for any reason, `notes-cli` still creates your note — it just skips the AI summary and prints the error to the console, so you never lose a transcript or note body because of an API problem.