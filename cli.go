package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/joho/godotenv"
)

type CLIOptions struct {
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
}

func RunInteractiveWizard() (CLIOptions, error) {
	var opts CLIOptions

	// Load defaults from environment variables
	opts.APIKey = os.Getenv("GROQ_API_KEY")
	opts.JoplinToken = os.Getenv("JOPLIN_TOKEN")
	opts.TranscriptPath = os.Getenv("TRANSCRIPT_PATH")
	opts.Model = "llama-3.3-70b-versatile"
	opts.Output = "notes.json"

	if opts.TranscriptPath == "" {
		opts.TranscriptPath = "transcript.txt" // default fallback
	}

	var configureOptional bool

	// Define the form steps
	form := huh.NewForm(
		// Group 1: Core Details
		huh.NewGroup(
			huh.NewInput().
				Title("Course Name").
				Description("What is the name of the course?").
				Value(&opts.Course).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("course name is required")
					}
					return nil
				}),
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
		),

		// Group 2: Transcript Path (Conditional)
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
			return opts.InputMethod != "file"
		}),

		// Group 3: Transcript Paste (Conditional)
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
			return opts.InputMethod != "paste"
		}),

		// Group 4: API Keys (Only show if missing or explicitly configuring)
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
			return opts.APIKey != ""
		}),

		// Group 5: Optional Settings Prompt
		huh.NewGroup(
			huh.NewConfirm().
				Title("Configure advanced settings?").
				Description("Joplin Sync, AI Model, Output Path").
				Value(&configureOptional),
		),

		// Group 6: Advanced Settings (Conditional)
		huh.NewGroup(
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
			return !configureOptional
		}),
	)

	err := form.Run()
	if err != nil {
		return opts, err
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
		envPath := getConfigPath()
		err := godotenv.Write(env, envPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not save config to .env: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "💾 Configuration saved to .env for future use!\n")
		}
	}

	return opts, nil
}
