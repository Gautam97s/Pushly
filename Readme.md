# Pushly Implementation Roadmap

Pushly will be implemented in small, testable phases. The first release targets Windows while keeping the core interfaces portable.

For detailed phase parts, acceptance criteria, and progress tracking, see [IMPLEMENTATION_PHASES.md](IMPLEMENTATION_PHASES.md).

## Phase 1: Local Git Core

- Initialize the Go module and CLI.
- Execute fixed, read-only Git operations through `os/exec`.
- Add command timeouts, structured errors, and unit tests.
- Support `status`, `diff`, `branch`, and `remote`.

## Phase 2: Repository and Scanner Layer

- Store configured parent folders with normalized absolute paths.
- Discover repositories with bounded depth and count limits.
- Skip symlinks and inaccessible paths without stopping the entire scan.
- Detect both normal repositories and worktree-style repositories.
- Validate that a requested path is inside an authorized repository root.
- Identify the repository root using Git rather than trusting the user-supplied folder.
- Add repository metadata: name, absolute path, current branch, remotes, and last scan time.
- Add machine-readable Git status output and parse it into structured models.
- Represent each changed path with its state: staged, unstaged, modified, added, deleted, renamed, or untracked.
- Preserve paths containing spaces, Unicode characters, and rename pairs.
- Create immutable status snapshots that can be compared before write operations.
- Detect stale snapshots when the repository changes after the phone reviewed it.
- Treat untracked files explicitly; normal `git diff` does not include them.
- Add a controlled untracked-file diff/read path for user-selected files without uploading source code during discovery.

### Phase 2 Status Model

The CLI currently returns raw Git text. Phase 2 converts that text into data the future mobile app and agent can use:

```text
RepositoryStatus
├── repository root
├── branch
├── upstream
├── ahead/behind counts
├── clean state
└── changes[]
	├── path
	├── original path, when renamed
	├── staged state
	├── worktree state
	└── untracked state
```

Git status should use a machine-readable format such as `--porcelain` with NUL-separated records. Human-readable output is for display only and must not be used as the parser contract.

### Phase 2 Acceptance Checks

- Scan a folder containing multiple repositories and nested repositories.
- Confirm symlinks cannot escape the configured parent folder.
- Confirm inaccessible folders are reported and do not abort the scan.
- Parse clean, modified, staged, deleted, renamed, and untracked files.
- Parse filenames containing spaces and rename pairs correctly.
- Reject repository paths outside configured folders.
- Confirm a changed repository produces a new status snapshot and invalidates the old one.
- Confirm discovery does not read or upload source-file contents.

## Phase 3: Local Windows Agent

- Add explicit file selection and safe staging.
- Implement commit and push using existing local Git authentication.
- Serialize operations per repository.
- Handle hooks, timeouts, bounded output, and stale-state failures.
- Add Windows background-process and configuration support.

## Phase 4: Secure Backend and Agent Protocol

- Add account and device authentication.
- Add short-lived QR/code pairing and revocation.
- Add authenticated outbound WebSocket connectivity.
- Validate operation IDs, expiry, nonces, repository scope, and allowlisted commands in the agent.
- Relay encrypted diff payloads without storing source content.
- Reconcile operation state after uncertain disconnects.

## Phase 5: Flutter Mobile MVP

- Add login, device, repository, status, changed-file, diff, commit, push, and result screens.
- Require explicit file selection before commit.
- Show actionable errors for stale state, authentication, hooks, conflicts, timeouts, and offline agents.

## Phase 6: Security and Release Hardening

- Test unauthorized devices, revoked pairings, replayed commands, invalid paths, symlink escapes, arbitrary command attempts, and oversized output.
- Test disconnects after commit and push.
- Document Windows installation and Git prerequisites.
- Defer macOS/Linux packaging and advanced development features until the MVP is stable.

## Current Slice

Phase 1 starts with a local CLI and a safe Git execution layer. Networking, mobile UI, and write operations will be added only after this local foundation is tested.
