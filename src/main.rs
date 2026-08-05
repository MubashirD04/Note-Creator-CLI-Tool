use clap::{Parser, Subcommand};
use colored::Colorize;
use std::fs::{self, File};
use std::io::{self, Read, Write};
use std::env;

mod config;
mod api;
use api::summarize_text;
use config::{save_config, load_config, get_config_path};

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

fn create_md_note(dir: &str, title: &str, body: &str, tags: Option<Vec<String>>) -> io::Result<()> {
    fs::create_dir_all(dir)?;

    let filename = format!("{}\\{}.md", dir, title);
    let mut file = File::create(&filename)?;

    let formatted_tags = match tags {
        Some(t) => format!("tags: [[{}]]", t.join(", ")),
        None => "tags: none".to_string(),
    };

    let content = format!("# {}\n\n{}\n\n{}\n", title, formatted_tags, body);
    file.write_all(content.as_bytes())?;

    println!("\nCreated: {}", filename.green());
    Ok(())
}

fn read_from_stdin() -> io::Result<String> {
    println!("{}","Paste your transcript below (Press Ctrl+D on Unix or Ctrl+Z on Windows then Enter to finish):".yellow());
    let mut buffer = String::new();
    io::stdin().read_to_string(&mut buffer)?;
    Ok(buffer)
}

fn uninstall_cli() -> io::Result<()> {
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
                println!("{}", "Generating summary with Groq...".yellow());
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
