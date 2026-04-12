package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
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
		fmt.Fprintf(os.Stderr, "  -o, --output string         JSON file path (default \"~/.notes-cli.json\")\n")
		fmt.Fprintf(os.Stderr, "--clear                     Clear existing notes file\n\n")
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
	fmt.Fprintf(os.Stderr, "  Groq-Powered Note Intelligence \n")
	fmt.Fprintln(os.Stderr, "--------------------------------------------")

	// 1. Core Identification
	course := flag.String("course", "", "Course name")
	title := flag.String("title", "", "Lecture title")

	// 2. API Keys & Auth
	apiKey := flag.String("api-key", "", "Groq API Key")
	joplinToken := flag.String("joplin-token", "", "Joplin Web Clipper Token")
	model := flag.String("model", "llama-3.3-70b-versatile", "Groq AI Model")

	// 3. Output & Management
	notesDefault := getNotesPath()
	output := flag.String("output", notesDefault, "JSON output path")
	clear := flag.Bool("clear", false, "Clear output file")

	// 4. Advanced Features
	askMode := flag.String("ask", "", "Ask a question about your notes")
	syncMode := flag.Bool("sync", false, "Repopulate notes.json from Joplin")

	// Support short flags
	flag.StringVar(course, "c", "", "Course name")
	flag.StringVar(title, "t", "", "Lecture title")
	flag.StringVar(apiKey, "k", "", "Groq API Key")
	flag.StringVar(joplinToken, "jt", "", "Joplin Token")
	flag.StringVar(model, "m", "llama-3.3-70b-versatile", "Groq AI Model")
	flag.StringVar(output, "o", notesDefault, "JSON output path")

	flag.Parse()

	// Check if data is being piped to stdin
	stat, _ := os.Stdin.Stat()
	isPiped := (stat.Mode() & os.ModeCharDevice) == 0
	isInteractive := !isPiped && *askMode == "" && !*syncMode && (*course == "" || *title == "")

	for {
		var transcript string
		if isInteractive {
			opts, err := RunInteractiveWizard()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Aborted: %v\n", err)
				os.Exit(1)
			}

			if opts.Action == "exit" {
				fmt.Fprintln(os.Stderr, "👋 Goodbye!")
				return
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

			if opts.Action == "upload" {
				if opts.InputMethod == "paste" {
					transcript = opts.TranscriptText
				} else if opts.InputMethod == "file" {
					transcriptBytes, err := os.ReadFile(opts.TranscriptPath)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error reading transcript file (%s): %v\n", opts.TranscriptPath, err)
						continue
					}
					transcript = string(transcriptBytes)
				}
			}

			if opts.Action == "ask" {
				*askMode = opts.AskQuery
				*syncMode = false
			} else if opts.Action == "sync" {
				*syncMode = true
				*askMode = ""
				fmt.Fprintln(os.Stderr, "") // Space before action
			} else {
				*syncMode = false
				*askMode = ""
			}
		}

		// Keys check
		if *apiKey == "" {
			*apiKey = os.Getenv("GROQ_API_KEY")
		}
		if *joplinToken == "" {
			*joplinToken = os.Getenv("JOPLIN_TOKEN")
		}

		storage := NewFileStorage(*output)

		if *clear {
			fmt.Fprintf(os.Stderr, "Sweep: Cleaning up output file...\n")
			storage.Clear()
			if !isInteractive {
				return
			}
			*clear = false
			continue
		}

		if *syncMode {
			fmt.Fprintf(os.Stderr, "🔄 Syncing notes from Joplin...\n")
			if *joplinToken == "" {
				fmt.Fprintln(os.Stderr, "Error: Joplin Token is required.")
			} else {
				jClient := NewJoplinClient(*joplinToken)
				folders, err := jClient.ListFolders()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				} else {
					existingNotesFile, _ := storage.LoadNotesFile()
					newNotesFile := NotesFile{Courses: make(map[string][]NoteEntry)}
					cachedCount := 0
					for _, folder := range folders {
						notes, _ := jClient.GetFolderNotes(folder.ID)
						for _, jNote := range notes {
							var notesBody interface{}
							var created string
							cached := false
							// Case-insensitive course name lookup
							var existingEntries []NoteEntry
							for cachedCourse, entries := range existingNotesFile.Courses {
								if strings.EqualFold(cachedCourse, folder.Title) {
									existingEntries = entries
									break
								}
							}

							for _, e := range existingEntries {
								if strings.EqualFold(e.Title, jNote.Title) {
									notesBody = e.Notes
									created = e.CreatedAt
									cached = true
									break
								}
							}
							if !cached {
								nb, _ := jClient.GetNoteBody(jNote.ID)
								notesBody = nb.Body
								created = time.UnixMilli(nb.CreatedTime).Format(time.RFC3339)
							} else {
								cachedCount++
							}
							newNotesFile.Courses[folder.Title] = append(newNotesFile.Courses[folder.Title], NoteEntry{
								Title:             jNote.Title,
								CreatedAt:         created,
								TranscriptSnippet: "Synced",
								Notes:             notesBody,
							})
						}
					}
					storage.SaveNotesFile(newNotesFile)
					fmt.Fprintf(os.Stderr, "✅ Sync Complete (%d cached)\n\n", cachedCount)
				}
			}
			if !isInteractive {
				return
			}
			*syncMode = false
			continue
		}

		if *askMode != "" {
			notesData, _ := storage.LoadNotesFile()
			filteredNotes := NotesFile{Courses: make(map[string][]NoteEntry)}
			qLower := strings.ToLower(*askMode)
			kws := strings.Fields(qLower)
			for cName, entries := range notesData.Courses {
				for _, entry := range entries {
					match := false
					content := strings.ToLower(entry.Title + " " + cName + " " + entry.TranscriptSnippet)
					for _, kw := range kws {
						if len(kw) > 2 && strings.Contains(content, kw) {
							match = true
							break
						}
					}
					if match {
						filteredNotes.Courses[cName] = append(filteredNotes.Courses[cName], entry)
					}
				}
			}
			cBytes, _ := json.MarshalIndent(filteredNotes, "", "  ")
			if len(cBytes) > 50 {
				groqClient := NewGroqClient(*apiKey, *model)
				answer, _ := groqClient.AnswerQuestion(string(cBytes), *askMode)
				fmt.Fprintf(os.Stdout, "\n--- ANSWER ---\n%s\n\n", answer)
			} else {
				fmt.Fprintln(os.Stderr, "No relevant notes found.")
			}
			if !isInteractive {
				return
			}
			*askMode = ""
			continue
		}

		// Default: Upload/Generate
		if *course == "" || *title == "" || *apiKey == "" {
			if isInteractive {
				continue
			}
			os.Exit(1)
		}

		if transcript == "" && !isInteractive {
			tb, _ := io.ReadAll(os.Stdin)
			transcript = string(tb)
		}

		if transcript != "" {
			statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
			statusMsg := fmt.Sprintf("%s %s > %s...", statusStyle.Render("🚀 Generating:"), *course, *title)

			// Simple Spinner Implementation
			done := make(chan bool)
			go func() {
				frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
				i := 0
				for {
					select {
					case <-done:
						return
					default:
						fmt.Fprintf(os.Stderr, "\r%s %s", frames[i], statusMsg)
						i = (i + 1) % len(frames)
						time.Sleep(100 * time.Millisecond)
					}
				}
			}()

			groqClient := NewGroqClient(*apiKey, *model)
			notesJSON, generateErr := groqClient.GenerateNotes(transcript)
			done <- true
			fmt.Fprintf(os.Stderr, "\r\033[2K") // Clear the spinner line

			if generateErr == nil {
				var notesObj map[string]interface{}
				json.Unmarshal([]byte(notesJSON), &notesObj)
				entry := NoteEntry{
					Title:             *title,
					CreatedAt:         time.Now().Format(time.RFC3339),
					TranscriptSnippet: "Generated",
					Notes:             notesObj,
				}
				storage.UpdateNotes(*course, entry)
				fmt.Fprintf(os.Stderr, "✅ Success: %s > %s\n\n", *course, *title)

				// Sync back to Joplin
				jClient := NewJoplinClient(*joplinToken)
				if jClient != nil {
					fId, _ := jClient.GetOrCreateFolder(*course)
					formatter := NewMarkdownFormatter()
					jClient.CreateNote(entry.Title, formatter.FormatNote(entry, *course), fId)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", generateErr)
			}
		}

		if !isInteractive {
			return
		}
		// Reset for next interaction
		*course = ""
		*title = ""
	}
}

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".env" // fallback to current dir
	}
	return filepath.Join(home, ".notes-cli.env")
}
func getNotesPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "notes.json" // fallback to current dir
	}
	return filepath.Join(home, ".notes-cli.json")
}
