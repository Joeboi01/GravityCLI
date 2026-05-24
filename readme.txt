# GravityCLI

A beautiful, interactive, and terminal-based Git & GitHub workflow client.

## Overview

GravityCLI is designed to simplify your daily Git and GitHub operations through an interactive Terminal User Interface (TUI). It wraps complex Git commands and GitHub API interactions into a stunning console graphics interface. 

It works seamlessly across **Windows**, **macOS**, and **Linux** on any terminal — including the standard Windows Command Prompt (`cmd.exe`), PowerShell, Windows Terminal, macOS Terminal, iTerm2, and Linux terminals.

## Features

- ⚡ **Interactive Dashboard:** A unified hub for all your operations.
- 🔑 **GitHub Authentication:** Securely connect your GitHub account via OAuth directly in the terminal.
- 📥 **Repository Management:** Search and clone existing repositories or create new ones directly from the terminal.
- 🌿 **Branch Management:** Switch between branches, create new branches, and manage your local workflow interactively.
- 🔀 **Pull Requests:** Create pull requests on GitHub without leaving your terminal.
- 📁 **Directory Browser & Commit Engine:** Navigate your files, stage changes interactively, and commit with ease.

## Installation

GravityCLI is distributed as a single standalone executable. You don't need Go, Node.js, or any other runtime installed on your machine to use it.

1. Locate the binary for your operating system in the `dist/` directory.
   - **Windows:** `gravity-windows-amd64.exe`
   - **Linux:** `gravity-linux-amd64`
   - **macOS (Intel):** `gravity-macos-amd64`
   - **macOS (Apple Silicon):** `gravity-macos-arm64`
2. Move the binary to a directory of your choice.
3. (Optional) Add the directory to your system's `PATH` environment variable so you can run `gravity` from anywhere.

### For Windows Users (`cmd.exe` / PowerShell)

GravityCLI has been specifically built and optimized to run flawlessly in standard Windows Command Prompt (`cmd.exe`) and legacy PowerShell. 

It automatically configures the correct Unicode encoding (UTF-8) and enables Virtual Terminal Processing, ensuring that all emojis, borders, colors, and UI elements render perfectly out-of-the-box.

Simply open `cmd.exe` or PowerShell, navigate to the folder containing the executable, and run it:
```cmd
gravity-windows-amd64.exe
```

## Usage

To start the main interactive dashboard, simply run the executable without any arguments:

```bash
gravity
```
*(Assuming you renamed the binary to `gravity` or `gravity.exe` and added it to your PATH)*

You can also use specific subcommands to jump directly to a feature:

- `gravity auth` - Authenticate with your GitHub account
- `gravity clone` - Clone a repository from GitHub interactively
- `gravity repo` - Create a new GitHub repository
- `gravity branches` - Switch and create local branches
- `gravity pr` - Create pull requests on GitHub
- `gravity nav` - Interactive Directory Browser & Git Cockpit

### Navigation & Controls

- Use the **Up/Down Arrow Keys** (`↑`/`↓`) or **k/j** to navigate menus and lists.
- Press **Enter** to select an action or confirm an input.
- Press **Esc** to cancel an action or go back.
- Press **q** or **Ctrl+C** to exit the application.
- When typing in input fields, use standard keyboard controls.
