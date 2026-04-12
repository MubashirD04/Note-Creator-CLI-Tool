# notes-cli

A high-performance Go CLI that transforms lecture transcripts into structured, detailed AI notes. Grouped by course, synced to Joplin, and searchable via a built-in AI Assistant.

## New in v2.0

- **AI Assistant (`--ask`):** Ask questions about your entire note library. "What are pointers used for?"
- **Full Joplin Sync (`--sync`):** Lost your `notes.json`? Repopulate it instantly from your Joplin notebooks.
- **Persistent Interactive Loop:** The CLI now stays open after syncing or asking questions, allowing for a continuous workflow.
- **Context Pruning:** Smart token management ensures you never hit Groq's TPM limits, even with thousands of notes.

## Key Features

- **Groq Llama 3.3 Power:** Uses the latest high-performance versatile models (128k context).
- **In-Depth Technical Notes:** Optimized prompts for exhaustive summaries, granular implementation steps, and detailed code examples.
- **Intelligent Q&A:** Query your notes directly. The AI Assistant uses context pruning to find relevant notes and answer your questions accurately.
- **Interactive Wizard:** Run the CLI without arguments to launch a guided, beautiful interactive UI!
- **Joplin Integration:** Automatic notebook discovery, folder creation, and high-quality Markdown syncing.
- **Auto-Config Persistence:** Wizard entries and API keys are automatically saved to `~/.notes-cli.env`.
- **One-File Database:** All your course notes in one `notes.json`, optimized for offline access and AI querying.

---

## Setup

```bash
# 1. Clone / place the project folder
cd notes-cli

# 2. Build the binary
make build

# 3. (Optional) install globally
make install
# → binary: /usr/local/bin/notes-cli
# → config: ~/.notes-cli.env
```

---

## Usage

### Interactive Mode (The Loop)

Simply run the CLI without any arguments to launch the full-featured interactive dashboard:

```bash
notes-cli
```

The wizard now supports a **persistent loop**:

1. **Sync from Joplin**: Repopulate your local database. (Goes back to menu)
2. **Ask a Question**: Query your notes library. (Goes back to menu)
3. **Generate Notes**: Upload a new transcript. (Finishes session)
4. **Exit**: Gracefully close the CLI.

### AI Assistant (Q&A)

Ask questions about your existing notes using your Groq API key:

```bash
notes-cli --ask "Explain the difference between Slices and Arrays in Go"
```

_Note: Uses smart context pruning to stay within Groq's Tokens Per Minute (TPM) limits._

### Full Joplin Sync

Repopulate your local `notes.json` from your Joplin notes:

```bash
notes-cli --sync
```

_Tip: Uses a "Delta Sync" strategy—skips redownloading notes already present in your local cache._

### Basic Note Generation

```bash
cat transcript.txt | notes-cli --course "Go" --title "Interfaces"
```

---

## All Flags

| Flag             | Short | Description                                        |
| ---------------- | ----- | -------------------------------------------------- |
| `--course`       | `-c`  | Course name grouping                               |
| `--title`        | `-t`  | Lecture title                                      |
| `--ask`          |       | Ask a question about your notes                    |
| `--sync`         |       | Repopulate local JSON from Joplin                  |
| `--api-key`      | `-k`  | Groq API key (Required if env not set)             |
| `--joplin-token` | `-jt` | Joplin Web Clipper Token                           |
| `--model`        | `-m`  | Groq AI Model (default: `llama-3.3-70b-versatile`) |
| `--output`       | `-o`  | Path to the JSON database                          |
| `--clear`        |       | Clear the notes JSON file                          |
| `--help`         | `-h`  | Display help message                               |

---

## Output format

Notes are stored in a grouped JSON structure and synced as rich Markdown to Joplin.

```json
{
  "courses": {
    "Go Bootcamp": [
      {
        "title": "Goroutines",
        "created_at": "2026-03-22T...",
        "notes": {
          "summary": "...",
          "key_concepts": [...],
          "detailed_notes": "...",
          "code_examples": [...]
        }
      }
    ]
  }
}
```

---

## Troubleshooting

- **Connection Refused (41184):** Ensure Joplin is running and **Web Clipper API** is enabled in settings.
- **TPM Limit (413/429):** Your library is large! The tool automatically prunes context, but if you still hit this, try a different model (`-m mixtral-8x7b-32768`).
- **Empty Database:** Start by generating a note or running `--sync` if you have notes in Joplin.

---

## License

MIT - Feel free to use and expand!
