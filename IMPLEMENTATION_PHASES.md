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

### What has been built

- Set up the Go project.
- Created the Pushly command-line program.
- Connected Pushly to the Git installed on the computer.
- Added a fixed list of safe read-only commands.
- Added commands to view:
  - Project status
  - File differences
  - Current branch
  - GitHub or other remote addresses
- Added a time limit so a stuck Git command does not run forever.
- Added useful error messages.
- Added tests for allowed and rejected commands.

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

### Current limits

- The output is still mostly Git's normal text.
- A normal diff does not show files that Git has never seen before.
- Pushly cannot commit or push yet.

---

## Phase 2: Finding Projects and Understanding Changes

**Status: Completed for the current implementation scope**

### What this phase means

Pushly now has the basic building blocks needed to find projects on the computer and understand what has changed inside them.

### Part 2.1: Understanding project changes

Pushly can identify:

- The current branch.
- The connected remote branch.
- Whether the project is clean or has changes.
- New files.
- Edited files.
- Deleted files.
- Renamed files.
- Files already prepared for a commit.
- Files that are changed but not prepared.
- File names containing spaces or special characters.

The information is converted into a consistent format so the future phone app can display it clearly instead of trying to understand raw terminal text.

### Part 2.2: Checking project locations

Pushly checks that:

- A selected location is really a Git project.
- The project is inside a folder the user approved.
- A file path cannot escape the approved project folder.
- A shortcut or link cannot secretly point outside the approved folder.
- Windows paths are handled consistently.

### Part 2.3: Approved folders

The user will choose which parent folders Pushly may inspect.

Pushly now has a folder manager that:

- Saves approved folders.
- Cleans up folder paths before saving them.
- Rejects missing or invalid folders.
- Rejects duplicate folders.
- Rejects folders that overlap with another approved folder.
- Allows an approved folder to be removed.

### Part 2.4: Finding Git projects

Pushly can search inside an approved folder and find Git projects.

The search:

- Can look inside nested folders.
- Has limits so it does not scan an entire computer by accident.
- Skips shortcuts and links.
- Skips folders it cannot access and reports them.
- Finds normal Git projects and Git worktrees.
- Does not read or upload project files just to find projects.
- Can be run again when the user wants a fresh search.

### Part 2.5: Remembering the reviewed state

When the user reviews a project, Pushly can create a record of what the project looked like at that moment.

Before a future commit, Pushly can check the project again. If something changed after the user reviewed it, Pushly will stop and ask the user to review the changes again. This helps prevent accidentally committing new work that the user never approved.

### Part 2.6: New files

Git's normal diff command does not show the contents of brand-new files. Pushly handles these separately:

- New files are shown in the change list.
- Their contents are not read during project discovery.
- A file is read only after the user specifically selects it.
- Large files are rejected by a size limit.
- Unsafe paths and links are rejected.
- File contents are not sent anywhere during discovery.

### What is confirmed

- [x] Pushly understands clean, edited, new, deleted, and renamed files.
- [x] It handles file names with spaces and special characters.
- [x] It can find projects inside nested folders.
- [x] Search limits work.
- [x] Links and inaccessible folders are handled safely.
- [x] Approved-folder checks work.
- [x] Project locations are checked using Git.
- [x] Selected files can be read safely within size limits.
- [x] Pushly can detect whether a reviewed project changed later.
- [x] Tests cover the status parser and project scanner.
- [ ] More Windows-specific tests are still useful.

---

## Phase 3: Windows Desktop Agent

**Status: Planned**

### What this phase means

We will turn the local tools into a background program that can safely make approved changes to a project on the Windows computer.

### What we will build

- Save the user's settings on Windows.
- Let the user choose specific files for a commit.
- Make sure selected files belong to the correct project.
- Commit only the selected files.
- Never include every changed file by accident.
- Accept and check a commit message.
- Push using the Git login already set up on the computer.
- Allow normal Git safety checks and hooks to run.
- Stop commands that take too long.
- Prevent Git from waiting forever for someone to type a password.
- Run only one operation at a time for each project.
- Check the project again immediately before changing it.
- Stop if the project is different from what the user reviewed.
- Return clear results such as waiting, running, successful, or failed.
- Keep the agent running in the background without an IDE.

### How we will know it works

- [ ] The user can choose individual files.
- [ ] Files not chosen by the user are not committed.
- [ ] Unsafe file paths are rejected.
- [ ] A project that changed after review is blocked.
- [ ] Commit and push use the computer's existing Git login.
- [ ] Git hook errors are shown clearly.
- [ ] Two conflicting actions cannot run on the same project at once.
- [ ] The agent works while the IDE is closed.

---

## Phase 4: Secure Connection Between Phone and Computer

**Status: Planned**

### What this phase means

We will connect the phone app and Windows agent through a secure online relay. The relay helps them communicate, but it should not store the user's project.

### What we will build

- Pushly account login.
- Computer registration.
- A secure connection started by the computer.
- Phone-to-computer pairing with a short-lived code or QR code.
- The ability to remove or block a paired device.
- Secure phone sessions.
- Clear message types for actions and results.
- Checks that confirm who is making a request and which project it targets.
- Protection against old requests being sent again.
- Protection against requests for projects the user does not own.
- Encrypted transfer of diffs so the relay cannot read them.
- Operation history without saving source code or passwords.
- Automatic reconnect and connection health checks.
- Careful handling when the connection drops during a commit or push.
- No blind repetition of a commit or push after an uncertain result.

### How we will know it works

- [ ] The agent connects securely to the relay.
- [ ] A phone can pair using a temporary code or QR code.
- [ ] A blocked device cannot send commands.
- [ ] Old or repeated requests are rejected.
- [ ] Requests for the wrong project are rejected.
- [ ] Random computer commands are rejected.
- [ ] The relay cannot read encrypted diffs.
- [ ] A connection failure cannot create duplicate commits or pushes.

---

## Phase 5: Flutter Mobile App

**Status: Planned**

### What this phase means

We will create the phone app that gives the user a simple way to control the authorized Windows agent.

### What the app will include

- Login.
- List of connected computers.
- Pairing screen.
- List of approved folders and projects.
- Project details.
- Current branch and change list.
- File selection.
- Diff viewing.
- Commit message box.
- Commit and push buttons.
- Progress while an action is running.
- Clear success and failure messages.
- Operation history and notifications.
- Helpful messages for offline computers, changed projects, login problems, Git errors, and conflicts.

### How we will know it works

- [ ] The user can pair with their computer.
- [ ] The user can choose a project.
- [ ] The app shows the correct project status.
- [ ] The user can choose exactly which files to commit.
- [ ] Diffs can be viewed safely.
- [ ] Commit and push results are clear and reliable.
- [ ] The relay does not permanently save source code.

---

## Phase 6: Final Safety and Release Checks

**Status: Planned**

### What this phase means

Before sharing Pushly with real users, we will test it against mistakes, unsafe requests, unusual files, and unreliable internet connections.

### What we will check

- Login and device security.
- Removing access from a device.
- Unsafe file paths and links.
- Access to someone else's computer or project.
- Expired or repeated requests.
- Attempts to run random commands.
- Very large requests or responses.
- Git hooks that take too long.
- Unusual file names and broken Git output.
- Someone changing the project locally while Pushly is using it.
- Internet disconnections during commit or push.
- Limits on repeated requests.
- Useful logs that do not contain passwords or source code.
- Safe Windows installation and updates.
- Clear setup instructions and Git requirements.
- Support for macOS and Linux later without rewriting the core.

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
