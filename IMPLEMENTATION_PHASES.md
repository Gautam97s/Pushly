# Pushly Implementation Phases

This document explains how Pushly is being built. It is written as a simple progress tracker so anyone can understand what the project does, what has been completed, and what is planned next.

## What We Want to Build

Pushly will let a developer use their phone to manage Git projects that are still stored on their own Windows computer.

The main goal is:

1. The developer leaves their computer and IDE as they are.
2. They open Pushly on their phone.
3. They choose their computer and a project.
4. They review the changes already present on the computer.
5. They choose which files to include.
6. They create a commit and push it to GitHub.
7. They receive the result on their phone.

The project is being built in small steps so each part can be tested before the next part is added.

## Status Meaning

- **Completed**: built and tested.
- **In progress**: currently being worked on.
- **Planned**: agreed for later, but not started.
- **Deferred**: intentionally saved for after the first usable version.

---

## Phase 1: Basic Git Commands

**Status: Completed**

### What this phase means

We created the first small Pushly command-line program. It can ask Git for basic information about a project.

### Try it

```powershell
go run ./cmd/pushly status .
go run ./cmd/pushly diff .
go run ./cmd/pushly branch .
go run ./cmd/pushly remote .
```

### What is confirmed

- [x] The Go project builds.
- [x] The commands work with a real Git project.
- [x] Unknown commands are rejected.
- [x] Git errors are shown to the user.
- [x] Tests pass.
- [x] Pushly does not run random computer commands.

---

## Phase 2: Finding Projects and Understanding Changes

**Status: Completed**

### What this phase means

Pushly has the building blocks needed to find projects on the computer and understand what has changed inside them.

### Part 2.1: Understanding project changes

Pushly can identify:

- The current branch and remote tracking branch.
- Whether the project is clean or has changes.
- New, edited, deleted, and renamed files.
- Files already prepared for a commit and files not yet prepared.
- File names containing spaces or special characters.

### Part 2.2: Approved folders & Project Discovery

Pushly checks that:

- The project is inside a folder the user approved.
- A shortcut or link cannot secretly point outside the approved folder.
- Windows paths are handled consistently.
- Searches inside approved folders find Git projects with bounded limits.

### Part 2.3: Remembering the reviewed state & New Files

- Pushly creates a snapshot record of what the project looked like when reviewed.
- If something changed after review, Pushly detects staleness and stops.
- Selected untracked files can be read safely up to a 1MB limit without escaping repo boundaries.

### What is confirmed

- [x] Pushly understands clean, edited, new, deleted, and renamed files.
- [x] It handles file names with spaces and special characters.
- [x] It can find projects inside nested folders with search limits.
- [x] Links and inaccessible folders are handled safely.
- [x] Approved-folder checks work.
- [x] Selected files can be read safely within size limits.
- [x] Pushly detects whether a reviewed project changed later.
- [x] Tests cover the status parser and project scanner.

---

## Phase 3: Windows Desktop Agent & Write Operations

**Status: Completed**

### What this phase means

We turned the local tools into a complete local agent coordinator that safely makes approved changes to projects on the Windows computer.

### What was built

- **Settings Persistence**: Saves and loads approved folders persistently in `%APPDATA%\Pushly\config.json`.
- **Selective File Staging**: Stages only explicitly chosen files, validating paths against directory traversal.
- **Snapshot-Protected Commits**: Validates commit messages and rejects committing if the working tree changed since review.
- **Git Push Support**: Non-interactive push (`GIT_TERMINAL_PROMPT=0`) preventing stuck background credential prompts.
- **Operation Serialization**: Per-repository mutex locking (`OperationQueue`) ensuring conflicting operations cannot collide.
- **Agent Coordinator**: Unified `Agent` engine orchestrating configuration, scanning, status, staging, committing, and pushing.
- **CLI Commands**: Subcommands for `folder`, `scan`, `status`, `status-json`, `diff`, `commit`, and `push`.

### Try it

```powershell
go run ./cmd/pushly folder list
go run ./cmd/pushly folder add C:\MyProjects
go run ./cmd/pushly scan
go run ./cmd/pushly commit . -m "My commit message" file1.go file2.go
go run ./cmd/pushly push . origin main
```

### What is confirmed

- [x] The user can configure approved folders with persistence.
- [x] The user can choose individual files for staging.
- [x] Files not chosen by the user are not committed.
- [x] Unsafe file paths and traversal attempts are rejected.
- [x] A project that changed after review is blocked with stale detection.
- [x] Commit and push use the computer's existing Git credentials non-interactively.
- [x] Two conflicting actions cannot run on the same project at once.
- [x] Agent tests and write operation tests pass cleanly.

---

## Phase 4: Secure Connection Between Phone and Computer

**Status: Completed**

### What this phase means

We created the secure online relay and communication protocol connecting the phone app and Windows agent without storing or exposing the user's source code.

### What was built

- **Protocol Envelope & Actions**: Standard typed JSON envelopes (`Message`, `CommandPayload`, `CommandResponsePayload`) with timestamp validation and expiration windows.
- **End-to-End Encryption (E2EE)**: Authenticated AES-256-GCM payload encryption ensuring plaintext diffs and file contents are never exposed to the relay.
- **Replay Protection**: Cryptographic nonce and sliding TTL replay cache rejecting duplicate and expired command requests.
- **Pairing & Session Management**: Generates 6-character short-lived pairing codes (5-minute TTL) with one-time use consumption, shared AES key derivation, and instant device revocation.
- **Relay Server**: Zero-knowledge HTTP/SSE streaming router dispatching commands to outbound connected desktop agents.
- **Agent Relay Client**: Reconnection loop with exponential backoff, automated command dispatch, and response encryption.
- **CLI Commands**: Added `pushly agent run` and `pushly agent pair`.

### Try it

```powershell
# In terminal 1 (start relay):
go run ./cmd/relay --port 8080

# In terminal 2 (start agent):
go run ./cmd/pushly agent run --relay http://localhost:8080

# In terminal 3 (generate pairing code):
go run ./cmd/pushly agent pair --relay http://localhost:8080
```

### What is confirmed

- [x] The agent connects securely to the relay via outbound stream.
- [x] A phone can pair using a temporary 6-character code.
- [x] A blocked or revoked device cannot send commands.
- [x] Old or repeated requests are rejected by replay protection.
- [x] Requests for unapproved projects are rejected by the agent coordinator.
- [x] Diffs and sensitive payloads are encrypted end-to-end with AES-256-GCM.
- [x] Full end-to-end pairing and encrypted command tests pass.

---

## Phase 5: Flutter Mobile App

**Status: Planned**

### What this phase means

We will create the mobile phone app that gives the user an intuitive interface to control their authorized Windows agent.

### What the app will include

- Login and computer selection.
- QR/code pairing screen.
- List of approved folders and projects.
- Branch & change list overview.
- File selection checkboxes.
- Diff viewer with syntax highlighting.
- Commit message composer with commit and push action buttons.
- Progress animations and clear operation results.

---

## Phase 6: Final Safety and Release Checks

**Status: Planned**

### What this phase means

Before sharing Pushly with real users, we will test it against edge cases, network drops, unusual repositories, and Windows installer packaging.

### Release requirement

Pushly will be ready for its first release when this complete story works:

1. The IDE is closed.
2. The agent is running on a Windows computer.
3. The phone is on another network.
4. The user pairs the phone with the computer.
5. Pushly finds an approved project.
6. The user reviews the changes.
7. The user chooses specific files.
8. The commit is created on the computer.
9. The changes are pushed to GitHub.
10. The phone receives a clear result.
11. No passwords or project files are permanently stored by Pushly's relay.

---

## Features for Later

These features are intentionally saved until the first version is reliable:

- Pulling changes.
- Switching and creating branches.
- Merging and stashing.
- Tags and reverting commits.
- Creating pull requests.
- Running tests and builds.
- Starting and stopping development servers.
- Viewing development logs.
- AI-generated commit messages and summaries.
- AI code review.
- Managing many computers.
- macOS and Linux background agents.
