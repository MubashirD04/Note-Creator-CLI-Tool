# Notes-CLI

A command-line interface tool written in Rust designed to create and manage Markdown notes. It seamlessly integrates with **Obsidian** vaults by outputting standard `.md` files complete with tags and optional AI-powered summaries powered by the Groq API.

---

## Features

* **Obsidian Integration:** Creates Markdown files with internal link tag formatting (`tags: [[tag1, tag2]]`).


* **AI Summarization:** Uses Groq API (`llama-3.1-8b-instant`) to auto-summarize input text or transcripts.


* **Configurable Paths:** Save your default Obsidian vault directory to avoid typing paths repeatedly.


* **Interactive Input:** Accept text arguments directly or paste long transcripts straight from `stdin`.


* **Self-Cleaner:** Built-in `uninstall` command to remove user config directories and binary files.



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