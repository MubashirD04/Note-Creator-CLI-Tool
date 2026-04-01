package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
	"strings"
)

type CLIOptions struct {
	Action         string
	AskQuery       string
	Course         string
	Title          string
	APIKey         string
	JoplinToken    string
	Model          string
	Output         string
	Clear          bool
	TranscriptPath string
	TranscriptText string
	InputMethod    string
	JoplinCourse   string
}

func RunInteractiveWizard() (CLIOptions, error) {
	var opts CLIOptions

	// Setup Blue Theme
	blue := lipgloss.Color("33") // Nice bright blue
	theme := huh.ThemeCharm()
	theme.Focused.Title = theme.Focused.Title.Foreground(blue)
	theme.Blurred.Title = theme.Blurred.Title.Foreground(blue)

	// Load defaults from environment variables
	opts.APIKey = os.Getenv("GROQ_API_KEY")
	opts.JoplinToken = os.Getenv("JOPLIN_TOKEN")
	opts.TranscriptPath = os.Getenv("TRANSCRIPT_PATH")
	opts.Model = "llama-3.3-70b-versatile"
	opts.Output = getNotesPath()

	if opts.TranscriptPath == "" {
		opts.TranscriptPath = "transcript.txt" // default fallback
	}

	// Fetch existing courses from Joplin AND local notes.json
	var courseOptions []huh.Option[string]
	courseMap := make(map[string]bool)

	jClient := NewJoplinClient(opts.JoplinToken)
	if jClient != nil {
		folders, err := jClient.ListFolders()
		if err == nil {
			for _, f := range folders {
				// Case-insensitive course name lookup
				found := false
				for existing := range courseMap {
					if strings.EqualFold(existing, f.Title) {
						found = true
						break
					}
				}
				if !found {
					courseOptions = append(courseOptions, huh.NewOption(f.Title, f.Title))
					courseMap[f.Title] = true
				}
			}
		} else {
			// Provide a gentle warning in the CLI console instead of crashing
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not connect to Joplin (%v).\n", err)
			fmt.Fprintf(os.Stderr, "   Ensure Joplin is running and Web Clipper is enabled to use course selection.\n\n")
		}
	}

	// Also fetch from local JSON if available
	storage := NewFileStorage(opts.Output)
	localNotes, err := storage.LoadNotesFile()
	if err == nil && localNotes.Courses != nil {
		for cName := range localNotes.Courses {
			if !courseMap[cName] {
				courseOptions = append(courseOptions, huh.NewOption(cName, cName))
				courseMap[cName] = true
			}
		}
	}
	courseOptions = append(courseOptions, huh.NewOption("Create New Course...", "NEW"))


	// Define the form steps
	form := huh.NewForm(
		// Group 1: Action Selection
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What would you like to do?").
				Options(
					huh.NewOption("Generate / Upload Notes", "upload"),
					huh.NewOption("Ask a Question", "ask"),
					huh.NewOption("Sync from Joplin", "sync"),
					huh.NewOption("Configure Settings", "settings"),
					huh.NewOption("Exit", "exit"),
				).
				Value(&opts.Action),
		),

		// Advanced Settings (Conditional)
		huh.NewGroup(
			huh.NewInput().
				Title("Groq API Key").
				Description("Your Groq API Key (saved to .env)").
				EchoMode(huh.EchoModePassword).
				Value(&opts.APIKey),
			huh.NewInput().
				Title("Joplin Token (Optional)").
				Description("Token for Joplin Web Clipper (saved to .env)").
				EchoMode(huh.EchoModePassword).
				Value(&opts.JoplinToken),
			huh.NewSelect[string]().
				Title("Groq AI Model").
				Options(
					huh.NewOption("LLaMA 3.3 70B", "llama-3.3-70b-versatile"),
					huh.NewOption("Mixtral 8x7B", "mixtral-8x7b-32768"),
					huh.NewOption("Gemma 7B", "gemma-7b-it"),
				).
				Value(&opts.Model),
			huh.NewInput().
				Title("Output Custom Path").
				Value(&opts.Output),
			huh.NewConfirm().
				Title("Clear Output File?").
				Description("Warning: This will clear the existing output file before saving.").
				Value(&opts.Clear),
		).WithHideFunc(func() bool {
			return opts.Action != "settings"
		}),

		// Ask Group (Conditional)
		huh.NewGroup(
			huh.NewInput().
				Title("Your Question").
				Description("What do you want to ask about your notes?").
				Value(&opts.AskQuery).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("question is required")
					}
					return nil
				}),
		).WithHideFunc(func() bool {
			return opts.Action != "ask"
		}),

		// Course Selection (Conditional)
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Course").
				Description("Choose an existing course from Joplin or create a new one.").
				Options(courseOptions...).
				Value(&opts.JoplinCourse),
		).WithHideFunc(func() bool {
			return (opts.Action != "upload" && opts.Action != "ask") || len(courseOptions) <= 1
		}),

		// New Course Name (Conditional)
		huh.NewGroup(
			huh.NewInput().
				Title("Course Name").
				Description("What is the name of the new course?").
				Value(&opts.Course).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("course name is required")
					}
					return nil
				}),
		).WithHideFunc(func() bool {
			return (opts.Action != "upload" && opts.Action != "ask") || (opts.JoplinCourse != "NEW" && opts.JoplinCourse != "")
		}),

		// Core Details
		huh.NewGroup(
			huh.NewInput().
				Title("Lecture Title").
				Description("What is the title of this lecture?").
				Value(&opts.Title).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("lecture title is required")
					}
					return nil
				}),
			huh.NewSelect[string]().
				Title("Transcript Input Method").
				Options(
					huh.NewOption("File Path", "file"),
					huh.NewOption("Direct Paste", "paste"),
				).
				Value(&opts.InputMethod),
		).WithHideFunc(func() bool {
			return opts.Action != "upload" && opts.Action != "ask" && opts.Action != "settings"
		}),

		// Transcript Path (Conditional)
		huh.NewGroup(
			huh.NewInput().
				Title("Transcript File Path").
				Description("Where is the transcript file located?").
				Value(&opts.TranscriptPath).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("transcript path is required")
					}
					if _, err := os.Stat(s); os.IsNotExist(err) {
						return fmt.Errorf("file does not exist: %s", s)
					}
					return nil
				}),
		).WithHideFunc(func() bool {
			return opts.Action != "upload" || opts.InputMethod != "file"
		}),

		// Transcript Paste (Conditional)
		huh.NewGroup(
			huh.NewText().
				Title("Transcript Content").
				Description("Paste the lecture transcript here.").
				Value(&opts.TranscriptText).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("transcript is required")
					}
					return nil
				}),
		).WithHideFunc(func() bool {
			return opts.Action != "upload" || opts.InputMethod != "paste"
		}),

		// API Keys (Conditional)
		huh.NewGroup(
			huh.NewInput().
				Title("Groq API Key").
				Description("Enter your Groq API Key (will be saved to .env)").
				EchoMode(huh.EchoModePassword).
				Value(&opts.APIKey).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("Groq API Key is required")
					}
					return nil
				}),
		).WithHideFunc(func() bool {
			return (opts.Action != "upload" && opts.Action != "ask") || opts.APIKey != ""
		}),

	).WithTheme(theme)

	err = form.Run()
	if err != nil {
		return opts, err
	}

	// Finalize course name
	if opts.JoplinCourse != "" && opts.JoplinCourse != "NEW" {
		opts.Course = opts.JoplinCourse
	}

	// Update .env file based on new keys or transcript path
	envUpdated := false
	envPath := getConfigPath()
	env, _ := godotenv.Read(envPath)
	if env == nil {
		env = make(map[string]string)
	}

	if opts.APIKey != "" && opts.APIKey != os.Getenv("GROQ_API_KEY") {
		env["GROQ_API_KEY"] = opts.APIKey
		envUpdated = true
	}
	if opts.JoplinToken != "" && opts.JoplinToken != os.Getenv("JOPLIN_TOKEN") {
		env["JOPLIN_TOKEN"] = opts.JoplinToken
		envUpdated = true
	}
	if opts.InputMethod == "file" && opts.TranscriptPath != "" && opts.TranscriptPath != os.Getenv("TRANSCRIPT_PATH") {
		env["TRANSCRIPT_PATH"] = opts.TranscriptPath
		envUpdated = true
	}

	if envUpdated {
		envPath = getConfigPath()
		err := godotenv.Write(env, envPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not save config: %v\n", err)
		} else {
			// Update current process env to stay in sync during loops
			for k, v := range env {
				os.Setenv(k, v)
			}
		}
	}

	return opts, nil
}
