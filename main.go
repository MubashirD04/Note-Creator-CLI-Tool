package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/joho/godotenv"
	"path/filepath"
)

func main() {
	envPath := getConfigPath()
	_ = godotenv.Load(envPath) // Setup .env file from binary location if it exists

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "NOTES CLI - Groq Edition\n")
		fmt.Fprintf(os.Stderr, "=========================\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  cat transcript.txt | ./notes-cli [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Required Flags:\n")
		fmt.Fprintf(os.Stderr, "  -c, --course string     Course name\n")
		fmt.Fprintf(os.Stderr, "  -t, --title string      Lecture title\n")
		fmt.Fprintf(os.Stderr, "  -k, --api-key string    Groq API Key (Required if GROQ_API_KEY env not set)\n\n")
		fmt.Fprintf(os.Stderr, "Optional Flags:\n")
		fmt.Fprintf(os.Stderr, "  -jt, --joplin-token string  Joplin Web Clipper Token (Enables Joplin sync)\n")
		fmt.Fprintf(os.Stderr, "  -m, --model string          Groq AI Model (default \"llama-3.3-70b-versatile\")\n")
		fmt.Fprintf(os.Stderr, "  -o, --output string         JSON file path (default \"notes.json\")\n")
		fmt.Fprintf(os.Stderr, "  --clear                     Clear existing notes file\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  cat transcript.txt | ./notes-cli -c \"Go\" -t \"Interfaces\"\n")
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

	// 1. Core Identification
	course := flag.String("course", "", "Course name")
	title := flag.String("title", "", "Lecture title")

	// 2. API Keys & Auth
	apiKey := flag.String("api-key", "", "Groq API Key")
	joplinToken := flag.String("joplin-token", "", "Joplin Web Clipper Token")
	model := flag.String("model", "llama-3.3-70b-versatile", "Groq AI Model")

	// 3. Output & Management
	output := flag.String("output", "notes.json", "JSON output path")
	clear := flag.Bool("clear", false, "Clear output file")

	// Support short flags
	flag.StringVar(course, "c", "", "Course name")
	flag.StringVar(title, "t", "", "Lecture title")
	flag.StringVar(apiKey, "k", "", "Groq API Key")
	flag.StringVar(joplinToken, "jt", "", "Joplin Token")
	flag.StringVar(model, "m", "llama-3.3-70b-versatile", "Groq AI Model")
	flag.StringVar(output, "o", "notes.json", "JSON output path")

	flag.Parse()

	// Check if data is being piped to stdin
	stat, _ := os.Stdin.Stat()
	isPiped := (stat.Mode() & os.ModeCharDevice) == 0

	var transcript string

	// Trigger interactive mode if not piped and missing required flags
	if !isPiped && (*course == "" || *title == "") {
		opts, err := RunInteractiveWizard()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Aborted: %v\n", err)
			os.Exit(1)
		}

		*course = opts.Course
		*title = opts.Title
		
		if opts.APIKey != "" {
			*apiKey = opts.APIKey
		}
		if opts.JoplinToken != "" {
			*joplinToken = opts.JoplinToken
		}
		*model = opts.Model
		
		if opts.Output != "" {
			*output = opts.Output
		}
		*clear = opts.Clear

		if opts.InputMethod == "paste" {
			transcript = opts.TranscriptText
		} else {
			// Read transcript from file path selected in wizard
			transcriptBytes, err := os.ReadFile(opts.TranscriptPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading transcript file (%s): %v\n", opts.TranscriptPath, err)
				os.Exit(1)
			}
			transcript = string(transcriptBytes)
		}
	}

	// Capture if keys were passed via flags
	passedApiKey := *apiKey
	passedJoplinToken := *joplinToken

	if *apiKey == "" {
		*apiKey = os.Getenv("GROQ_API_KEY")
	}
	if *joplinToken == "" {
		*joplinToken = os.Getenv("JOPLIN_TOKEN")
	}

	// If both keys were provided via flags, save them to .env to avoid repetition
	if passedApiKey != "" && passedJoplinToken != "" {
		// Try to read existing env to avoid wiping other variables (like JOPLIN_PORT)
		envPath := getConfigPath()
		env, _ := godotenv.Read(envPath)
		if env == nil {
			env = make(map[string]string)
		}

		updated := false
		if passedApiKey != "" {
			env["GROQ_API_KEY"] = passedApiKey
			updated = true
		}
		if passedJoplinToken != "" {
			env["JOPLIN_TOKEN"] = passedJoplinToken
			updated = true
		}

		if updated {
			envPath := getConfigPath()
			err := godotenv.Write(env, envPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not save config to .env: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "💾 Configuration saved to .env for future use!\n")
			}
		}
	}

	if *apiKey != "" && *joplinToken != "" {
		fmt.Fprintf(os.Stderr, "✅ Configuration Loaded: Groq AI & Joplin Sync active.\n\n")
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

	if transcript == "" {
		// Read transcript from stdin
		transcriptBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		
		transcript = string(transcriptBytes)
		if transcript == "" {
			fmt.Fprintln(os.Stderr, "Error: transcript is empty. Please pipe a transcript to stdin or use interactive mode.")
			os.Exit(1)
		}
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
	groqClient := NewGroqClient(*apiKey, *model)
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
	jClient := NewJoplinClient(*joplinToken)
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

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".env" // fallback to current dir
	}
	return filepath.Join(home, ".notes-cli.env")
}
