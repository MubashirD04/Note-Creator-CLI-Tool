use clap::{Parser, Subcommand};
use colored::Colorize;
use std::fs::{self, File};
use std::io::{self, Read, Write};
use std::env;
use std::path::Path;

mod config;
mod api;
use api::summarize_text;
use config::{save_config, load_config, get_config_path, clear_groq_key, set_groq_key};

//------CLI
#[derive(Parser)]
#[command(
    name = "Notes-CLI",
    version = "2.0",
    about = "Rust version of Notes-CLI with obsidian integration"
)]
struct Cli {
    #[command(subcommand)]
    command: Commands,

    #[arg(short, long)]
    list: bool,
}

#[derive(Subcommand)]
enum Commands {
    /// Add a new note
    Add {
        title: String,
        body: Option<String>,

        #[arg(short, long, value_delimiter = ',', num_args = 1..=5)]
        tags: Option<Vec<String>>,

        #[arg(short, long)]
        path: Option<String>,

        #[arg(short, long)]
        summarize: bool,
    },

    /// Set and persist the default output path
    SetPath { path: String },

    /// Explicitly set and persist your Groq API key (no interactive prompt required)
    SetKey { key: String },

    /// Remove the stored Groq API key from the OS keychain
    ClearKey,

    /// Completely uninstall notes-cli and remove user configurations
    Uninstall,
}

//------HELPERS
fn list_md_files(dir: &str) -> io::Result<()> {
    println!("\nMarkdown files in {}:", dir.blue());

    for entry in fs::read_dir(dir)? {
        let path = entry?.path();

        if path.is_file() && path.extension().and_then(|s| s.to_str()) == Some("md") {
            println!(" - {}", path.file_name().unwrap().to_string_lossy());
        }
    }
    Ok(())
}

fn sanitize_filename(title: &str) -> String {
    let cleaned: String = title
        .chars()
        .map(|c| match c {
            '/' | '\\' | ':' | '*' | '?' | '"' | '<' | '>' | '|' => '-',
            c if c.is_control() => ' ',
            c => c,
        })
        .collect();

    let trimmed = cleaned.trim().trim_matches('.');

    if trimmed.is_empty() {
        "untitled".to_string()
    } else {
        trimmed.to_string()
    }
}

fn sanitize_tags(tags: Option<Vec<String>>) -> Option<Vec<String>> {
    let cleaned: Vec<String> = tags?
        .into_iter()
        .map(|tag| tag.trim().to_string())
        .filter(|tag| !tag.is_empty())
        .collect();

    if cleaned.is_empty() {
        None
    } else {
        Some(cleaned)
    }
}

fn create_md_note(dir: &str, title: &str, body: &str, tags: Option<Vec<String>>) -> io::Result<()> {
    fs::create_dir_all(dir)?;

    let safe_title = sanitize_filename(title);
    let filename = Path::new(dir).join(format!("{}.md", safe_title));
    let mut file = File::create(&filename)?;

    let formatted_tags = match tags {
        Some(t) => {
            let wikilinks: Vec<String> = t.iter().map(|tag| format!("[[{}]]", tag)).collect();
            format!("tags: {}", wikilinks.join(", "))
        }
        None => "tags: none".to_string(),
    };

    let content = format!("# {}\n\n{}\n\n{}\n", title, formatted_tags, body);
    file.write_all(content.as_bytes())?;

    println!("\nCreated: {}", filename.display().to_string().green());
    Ok(())
}

fn read_from_stdin() -> io::Result<String> {
    println!("{}","Paste your transcript below (Press Ctrl+D on Unix or Ctrl+Z on Windows then Enter to finish):".yellow());
    let mut buffer = String::new();
    io::stdin().read_to_string(&mut buffer)?;
    Ok(buffer)
}

fn uninstall_cli() -> io::Result<()> {
    if let Err(e) = clear_groq_key() {
        eprintln!("Note: Could not clear keyring entry {}", e);
    }

    if let Ok(config_path) = get_config_path() {
        if let Some(config_dir) = config_path.parent() {
            if config_dir.exists() {
                fs::remove_dir_all(config_dir)?;
                println!("{}", "Cleared notes-cli configuration directory.".yellow());
            }
        }
    }

    let current_exe = env::current_exe()?;
    println!("Removing executable at: {}", current_exe.display());

    #[cfg(target_os = "windows")]
    {
        let temp_exe = current_exe.with_extension("exe.old");
        fs::rename(&current_exe, &temp_exe)?;
        fs::remove_file(temp_exe)?;
    }

    #[cfg(not(target_os = "windows"))]
    {
        fs::remove_file(current_exe)?;
    }

    println!("{}", "Notes-CLI has been successfully uninstalled.".green().bold());
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn sanitize_filename_replaces_illegal_characters() {
        assert_eq!(sanitize_filename("a/b\\c:d*e?f\"g<h>i|j"), "a-b-c-d-e-f-g-h-i-j");
    }

    #[test]
    fn sanitize_filename_strips_control_characters_to_space() {
        let with_control = format!("title{}end", '\u{0007}');
        assert_eq!(sanitize_filename(&with_control), "title end");
    }

    #[test]
    fn sanitize_filename_trims_whitespace_and_dots() {
        assert_eq!(sanitize_filename("  My Note...  "), "My Note");
    }

    #[test]
    fn sanitize_filename_falls_back_to_untitled_when_empty_after_cleanup() {
        assert_eq!(sanitize_filename("   ...   "), "untitled");
        assert_eq!(sanitize_filename(""), "untitled");
        assert_eq!(sanitize_filename("   "), "untitled");
    }

    #[test]
    fn sanitize_filename_does_not_treat_replaced_separators_as_empty() {
        // '/' becomes '-', which is a valid filename character on its own,
        // so this must NOT fall back to "untitled".
        assert_eq!(sanitize_filename("///"), "---");
    }

    #[test]
    fn sanitize_filename_preserves_normal_titles() {
        assert_eq!(sanitize_filename("Weekly Standup Notes"), "Weekly Standup Notes");
    }

    #[test]
    fn sanitize_tags_trims_and_drops_empty_entries() {
        let tags = vec![" work ".to_string(), "".to_string(), "  ".to_string(), "urgent".to_string()];
        let result = sanitize_tags(Some(tags));
        assert_eq!(result, Some(vec!["work".to_string(), "urgent".to_string()]));
    }

    #[test]
    fn sanitize_tags_returns_none_when_all_tags_are_blank() {
        let tags = vec!["  ".to_string(), "".to_string()];
        assert_eq!(sanitize_tags(Some(tags)), None);
    }

    #[test]
    fn sanitize_tags_returns_none_when_input_is_none() {
        assert_eq!(sanitize_tags(None), None);
    }

    #[test]
    fn create_md_note_writes_expected_content_with_tags() {
        let dir = tempdir().unwrap();
        let dir_path = dir.path().to_str().unwrap();

        create_md_note(dir_path, "My Title", "Body text", Some(vec!["a".to_string(), "b".to_string()])).unwrap();

        let content = fs::read_to_string(Path::new(dir_path).join("My Title.md")).unwrap();
        assert!(content.contains("# My Title"));
        assert!(content.contains("tags: [[a]], [[b]]"));
        assert!(content.contains("Body text"));
    }

    #[test]
    fn create_md_note_writes_none_when_no_tags() {
        let dir = tempdir().unwrap();
        let dir_path = dir.path().to_str().unwrap();

        create_md_note(dir_path, "Untagged", "Body", None).unwrap();

        let content = fs::read_to_string(Path::new(dir_path).join("Untagged.md")).unwrap();
        assert!(content.contains("tags: none"));
    }

    #[test]
    fn create_md_note_creates_missing_output_directory() {
        let dir = tempdir().unwrap();
        let nested = dir.path().join("nested").join("output");
        let nested_str = nested.to_str().unwrap();

        create_md_note(nested_str, "Note", "Body", None).unwrap();

        assert!(nested.join("Note.md").exists());
    }

    #[test]
    fn create_md_note_sanitizes_unsafe_title_for_filename() {
        let dir = tempdir().unwrap();
        let dir_path = dir.path().to_str().unwrap();

        create_md_note(dir_path, "Q1/Q2 Report", "Body", None).unwrap();

        assert!(Path::new(dir_path).join("Q1-Q2 Report.md").exists());
    }

    #[test]
    fn list_md_files_only_reports_markdown_files() -> io::Result<()> {
        let dir = tempdir().unwrap();
        fs::write(dir.path().join("note.md"), "content")?;
        fs::write(dir.path().join("ignore.txt"), "content")?;

        // Just verify it runs without error for a directory containing
        // both markdown and non-markdown files.
        list_md_files(dir.path().to_str().unwrap())
    }

    #[test]
    fn list_md_files_errors_on_missing_directory() {
        let result = list_md_files("Z:/definitely/does/not/exist/hopefully");
        assert!(result.is_err());
    }
}

#[tokio::main]
async fn main() -> io::Result<()> {
    let cli = Cli::parse();

    match cli.command {
        Commands::SetPath { path } => {
            let mut config = load_config();
            config.destination_path = path.clone();
            save_config(&config)?;
            println!("Default path saved as: {}", path.green());
        }
        Commands::SetKey { key } => {
            if let Err(e) = set_groq_key(&key) {
                eprintln!("{}: {}", "Failed to save API key".red(), e);
            }
        }
        Commands::ClearKey => {
            if let Err(e) = clear_groq_key() {
                eprintln!("{}: {}", "Failed to clear API key".red(), e);
            } else {
                println!("{}", "Stored Groq API key cleared.".green());
            }
        }
        Commands::Add {
            title,
            body,
            tags,
            path,
            summarize,
        } => {
            let output_dir = path.unwrap_or_else(|| load_config().destination_path);

            let mut content_body = match body {
                Some(b) => b,
                None => read_from_stdin()?,
            };

            if summarize {
                println!("{}", "\nGenerating summary with Groq...".yellow());
                match summarize_text(&content_body).await {
                    Ok(summary) => {
                        content_body = format!(
                            "## Summary\n{}\n\n---\n\n## Transcript\n{}", 
                            summary, 
                            content_body
                        );
                    }
                    Err(e) => {
                        eprintln!("{}: {}", "Failed to generate summary".red(), e);
                    }
                }
            }

            let tags = sanitize_tags(tags);

            create_md_note(&output_dir, &title, &content_body, tags)?;

            if cli.list {
                list_md_files(&output_dir)?;
            }
        }
        Commands::Uninstall => {
            println!("{}", "Are you sure you want to uninstall Notes-CLI? (y/N):".red());
            let mut confirm = String::new();
            io::stdin().read_line(&mut confirm)?;

            if confirm.trim().eq_ignore_ascii_case("y") {
                uninstall_cli()?;
            } else {
                println!("Uninstall cancelled.");
            }
        }
    }

    Ok(())
}