# notes-cli

A fast Go CLI that takes a lecture transcript from stdin, generates structured AI notes using Groq, and appends them to a JSON file grouped by course.

## Key Features

- **Groq Llama 3.3 Power:** Uses the latest high-performance versatile models.
- **Joplin Integration:** Automatic notebook creation and Markdown note syncing.
- **Smart Note Templates:** Notes include a **Table of Contents**, metadata, and tags.
- **Safety Measures:** Size warnings for large transcripts and atomic JSON writes.
- **One-File Database:** All your course notes in one `notes.json`, easily queryable with `jq`.


---

## Requirements

- [Go 1.22+](https://go.dev/dl/)
- A [Groq API key](https://console.groq.com/keys)
- (Optional) [Joplin](https://joplinapp.org/) for syncing notes via Web Clipper API

---

## Setup

```bash
# 1. Clone / place the project folder
cd notes-cli

# 2. Download dependencies
go mod tidy

# 3. Create a .env file
# Add your API keys:
echo "GROQ_API_KEY=your_key_here" > .env
echo "JOPLIN_TOKEN=your_joplin_token_here" >> .env

# 4. Build the binary
make build
# → produces ./notes-cli

# 5. (Optional) install globally
make install
# → copies to /usr/local/bin/notes-cli
```

---

## Usage

### Set your API key (once)

```bash
export GROQ_API_KEY=gsk-...
```

Or pass it per-command with `--api-key`.

### Basic usage — pipe a transcript

```bash
# Example Run
cat transcript.txt | ./notes-cli --course "Go Concurrency" --title "WaitGroups"
```

### Reset the notes database

```bash
./notes-cli --clear
```

**What happens next?**
1.  **CLI Banner** is displayed.
2.  **Groq Llama 3.3** processes the transcript.
3.  **JSON Entry** is appended to `notes.json`.
4.  **Joplin Notebook** ("Go Concurrency") is created/found.
5.  **Joplin Note** is created with a rich Markdown format!

### Specify a custom notes file

```bash
cat transcript.txt | ./notes-cli \
  -c "Go Bootcamp" \
  -t "Channels" \
  -o ~/notes/go_bootcamp.json
```

### Paste directly (multi-line, end with Ctrl+D)

```bash
./notes-cli --course "Go Bootcamp" --title "Interfaces"
# paste your transcript here...
# press Ctrl+D when done
```

### All flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--course` | `-c` | *(required)* | Course name — notes are grouped under this |
| `--title` | `-t` | *(required)* | Video/lecture title |
| `--output` | `-o` | `notes.json` | Path to the JSON notes file |
| `--api-key` | `-k` | `$GROQ_API_KEY` | Groq API key |
| `--help` | `-h` | `false` | Display help message |
| `--clear` | | `false` | Clear the notes JSON file |

---

## Output format

Notes are saved to a single JSON file with this structure:

```json
{
  "courses": {
    "Go Bootcamp": [
      {
        "title": "Goroutines and WaitGroups",
        "created_at": "2026-03-22T10:30:00+02:00",
        "transcript_snippet": "In this lecture we'll cover...",
        "notes": {
          "summary": "This lecture covered goroutines as lightweight threads...",
          "key_concepts": [
            "Goroutines are cheap — thousands can run concurrently",
            "sync.WaitGroup tracks in-flight goroutines",
            "go keyword spawns a goroutine"
          ],
          "detailed_notes": "Goroutines are Go's concurrency primitive...",
          "code_examples": [
            "var wg sync.WaitGroup\nwg.Add(1)\ngo func() { defer wg.Done(); ... }()\nwg.Wait()"
          ],
          "action_items": [
            "Practice building a worker pool with goroutines",
            "Read the sync package docs"
          ]
        }
      }
    ]
  }
}
```

Each new lecture is **appended** to its course — existing notes are never overwritten. The file is written atomically (via a temp file + rename) so it is never corrupted.

---

## Tips

- **Udemy captions** — open DevTools → Network tab → filter `.vtt` while watching a video, copy the caption text and pipe it in.
- **Multiple courses** — use the same `notes.json` file with different `--course` values; each course gets its own array.
- **Review your notes** — use `jq` to pretty-print: `jq '.courses["Go Bootcamp"][] | .notes.summary' notes.json`
- **Uninstall** — remove the global binary with `sudo make uninstall`