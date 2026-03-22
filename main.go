package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // Setup .env file if it exists

	// Custom Usage/Help
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "NOTES CLI - Groq Edition\n")
		fmt.Fprintf(os.Stderr, "=========================\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  cat transcript.txt | ./notes-cli [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  cat transcript.txt | ./notes-cli -c \"Go Bootcamp\" -t \"Interfaces\"\n")
		fmt.Fprintf(os.Stderr, "  ./notes-cli --clear\n")
	}

	// Startup Banner (ASCII Style)
	banner := `
  _   _  ____  _____ _____  ____     ____ _     ___ 
 | \ | |/ __ \|_   _|  ___|/ ___|   / ___| |   |_ _|
 |  \| | |  | | | | | |__  \___ \  | |   | |    | | 
 | |\  | |__| | | | |  __|  ___) | | |___| |___ | | 
 |_| \_|\____/  |_| |_____| ____/   \____|_____|___|
                                                    
`
	fmt.Fprintln(os.Stderr, banner)
	fmt.Fprintf(os.Stderr, "  🚀 Groq-Powered Note Intelligence 🚀\n")
	fmt.Fprintln(os.Stderr, "--------------------------------------------")

	course := flag.String("course", "", "Course name (required for notes)")
	title := flag.String("title", "", "Video/lecture title (required for notes)")
	output := flag.String("output", "notes.json", "Path to the JSON notes file")
	apiKey := flag.String("api-key", "", "Groq API key")
	clear := flag.Bool("clear", false, "Clear the contents of the notes file")

	// Support short flags
	flag.StringVar(course, "c", "", "Course name (short)")
	flag.StringVar(title, "t", "", "Video/lecture title (short)")
	flag.StringVar(output, "o", "notes.json", "Path to the JSON notes file (short)")
	flag.StringVar(apiKey, "k", "", "Groq API key (short)")

	flag.Parse()

	if *apiKey == "" {
		*apiKey = os.Getenv("GROQ_API_KEY")
	}

	storage := NewFileStorage(*output)

	if *clear {
		fmt.Fprintf(os.Stderr, "🧹 Clearing %s...\n", *output)
		err := storage.Clear()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Successfully cleared %s\n", *output)
		os.Exit(0)
	}

	if *course == "" || *title == "" {
		fmt.Fprintln(os.Stderr, "Error: --course and --title are required.")
		flag.Usage()
		os.Exit(1)
	}

	if *apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: GROQ_API_KEY environment variable or --api-key flag is required.")
		os.Exit(1)
	}

	// Read transcript from stdin
	transcriptBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	transcript := string(transcriptBytes)
	if transcript == "" {
		fmt.Fprintln(os.Stderr, "Error: transcript is empty. Please pipe a transcript to stdin.")
		os.Exit(1)
	}

	// (A) Token/Character Count Warning
	charCount := len(transcript)
	fmt.Fprintf(os.Stderr, "Transcript size: %d characters\n", charCount)
	if charCount > 50000 {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: Large transcript detected. This might exceed API limits.\n")
	}

	snippetLen := 50
	if len(transcript) < snippetLen {
		snippetLen = len(transcript)
	}
	snippet := transcript[:snippetLen] + "..."

	fmt.Fprintf(os.Stderr, "Generating notes for: %s\n", *title)

	// Call Groq API
	groqClient := NewGroqClient(*apiKey)
	notesJSONStr, err := groqClient.GenerateNotes(transcript)
	if err != nil {
		// (B) Prettier Error Explanations
		fmt.Fprintf(os.Stderr, "\n❌ FAILED TO GENERATE NOTES\n")
		fmt.Fprintf(os.Stderr, "-------------------------------\n")
		fmt.Fprintf(os.Stderr, "%v\n", err)
		fmt.Fprintf(os.Stderr, "-------------------------------\n")
		os.Exit(1)
	}

	var notesObj map[string]interface{}
	if err := json.Unmarshal([]byte(notesJSONStr), &notesObj); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing Groq response as JSON: %v\nResponse: %s\n", err, notesJSONStr)
		os.Exit(1)
	}

	entry := NoteEntry{
		Title:             *title,
		CreatedAt:         time.Now().Format(time.RFC3339),
		TranscriptSnippet: snippet,
		Notes:             notesObj,
	}

	// Read and update the notes file
	err = storage.UpdateNotes(*course, entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating notes file: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Successfully appended notes to %s\n", *output)

	// Joplin Sync logic
	jClient := NewJoplinClient()
	if jClient != nil {
		fmt.Fprintf(os.Stderr, "Syncing to Joplin...\n")
		folderId, err := jClient.GetOrCreateFolder(*course)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get or create Joplin folder: %v\n", err)
		} else {
			formatter := NewMarkdownFormatter()
			mdBody := formatter.FormatNote(entry, *course)
			err = jClient.CreateNote(entry.Title, mdBody, folderId)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to sync note to Joplin: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Successfully synced note '%s' to Joplin!\n", entry.Title)
			}
		}
	}
}
