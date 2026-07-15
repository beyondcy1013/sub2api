# Fork Setup Guide

## Why Fork?

Forking `Wei-Shaw/sub2api` on GitHub gives you a **remote backup** of every local commit. The current workflow relies on local bundle files under `/home/third_party/upgrade-backups/` for recovery — a fork moves that safety net to GitHub's servers.

**What changes with a fork:**
- Local commits are pushed to your fork, giving a remote history.
- Upstream sync becomes `git fetch upstream && git merge upstream/main` (same conflict resolution).
- No more fragile `git bundle` for emergency recovery.

**What does NOT change:**
- Conflict count and resolution difficulty stay the same (customizations still clash with upstream).
- The merge / reapply / test / deploy flow is identical.

## Step-by-Step Setup

### Prerequisites

- GitHub account with a personal access token (classic) that has `repo` scope.
- Token stored securely (not in code).

### 1. Fork the upstream repo on GitHub

Do this in the GitHub web UI (or via API once token is provided):
- Source: `https://github.com/Wei-Shaw/sub2api`
- Fork target: `<your-username>/sub2api`

### 2. Configure remotes

```bash
cd /home/third_party/sub2api

# Rename current origin (upstream) to 'upstream'
git remote rename origin upstream

# Add your fork as the new 'origin'
# Replace YOUR_USERNAME and YOUR_TOKEN
git remote add origin https://YOUR_USERNAME:YOUR_TOKEN@github.com/YOUR_USERNAME/sub2api.git

# Verify
git remote -v
# origin    https://YOUR_USERNAME:...@github.com/YOUR_USERNAME/sub2api.git (fetch)
# origin    https://YOUR_USERNAME:...@github.com/YOUR_USERNAME/sub2api.git (push)
# upstream  https://github.com/Wei-Shaw/sub2api.git (fetch)
# upstream  https://github.com/Wei-Shaw/sub2api.git (push)
```

### 3. Push local main to your fork

```bash
git push origin main
```

### 4. Create a local-customization branch (optional but recommended)

```bash
git checkout -b local-customizations
git push -u origin local-customizations
```

Keep `main` as a clean upstream mirror and do customization work on `local-customizations`.

## New Upgrade Flow (with fork)

```bash
# Fetch upstream changes
git fetch upstream

# Merge upstream into local main (or local-customizations branch)
git merge --no-ff upstream/main

# Resolve conflicts (same as before)
# Run tests
# Build, deploy, verify

# Push updated history to your fork
git push origin main
# or: git push origin local-customizations
```

## Token Management

Store the token in an env file, NOT in the remote URL directly:

```bash
# Better: use credential helper or env var
git config credential.helper store
# Or use ~/.netrc
# machine github.com login YOUR_USERNAME password YOUR_TOKEN
```

## Proxy Note

This machine's git uses proxy `127.0.0.1:7890`. The fork push/fetch will go through the same proxy:

```bash
# Ensure git proxy is set
git config --global http.proxy http://127.0.0.1:7890
git config --global https.proxy http://127.0.0.1:7890
# BUT clear conflicting env vars for git commands:
export http_proxy="" https_proxy="" HTTP_PROXY="" HTTPS_PROXY=""
```
