# Fork Setup Guide

## Current Configuration (Completed 2026-07-16)

Both `sub2api` and `sub2freeApi` are now backed by a GitHub fork.

- **Fork repo**: `https://github.com/beyondcy1013/sub2api`
- **Upstream**: `https://github.com/Wei-Shaw/sub2api`
- **GitHub account**: `beyondcy1013`
- **Auth**: gh CLI configured with PAT token

### Remote Layout (both repos)

```
origin    -> beyondcy1013/sub2api (fork, push here)
upstream  -> Wei-Shaw/sub2api (source, fetch only)
```

### Branch Layout

- **main** (on sub2api repo): full commit history with local customizations, 24+ commits ahead of upstream
- **sub2freeApi** (on fork): squashed clean branch, sub2freeApi-specific customizations on v0.1.156

## New Upgrade Flow

```bash
cd /home/third_party/sub2api   # or /home/third_party/sub2freeApi

# Clear conflicting env proxies (IMPORTANT - env proxy 12111 causes 407)
export http_proxy="" https_proxy="" HTTP_PROXY="" HTTPS_PROXY="" all_proxy="" ALL_PROXY=""
export HTTPS_PROXY="http://127.0.0.1:7890" HTTP_PROXY="http://127.0.0.1:7890"

# 1. Fetch upstream changes
git fetch upstream --no-tags

# 2. Merge (resolve conflicts as before)
git stash push -m "pre-upgrade-$(date +%s)"   # if dirty
git merge --no-ff upstream/main
git stash pop --index                           # if stashed

# 3. Resolve conflicts, verify customizations, test, build, deploy

# 4. Push to fork (remote backup)
git push origin main
```

Or use the automated script:
```bash
./scripts/upgrade-from-fork.sh /home/third_party/sub2api
```

## Recovery

If local state is lost, restore from fork:
```bash
git clone https://github.com/beyondcy1013/sub2api.git
cd sub2api
git remote add upstream https://github.com/Wei-Shaw/sub2api.git
```

For sub2freeApi:
```bash
git clone -b sub2freeApi https://github.com/beyondcy1013/sub2api.git sub2freeApi
cd sub2freeApi
git remote rename origin upstream  # careful: clone makes origin the fork
git remote add upstream https://github.com/Wei-Shaw/sub2api.git
```

## Proxy Notes

This machine uses two HTTP proxies:
- `127.0.0.1:7890` — working proxy for GitHub (used by git config)
- `127.0.0.1:12111` — env proxy that returns 407 for git operations

**Always clear env proxy vars before git operations**, then set the working one:
```bash
export http_proxy="" https_proxy="" HTTP_PROXY="" HTTPS_PROXY=""
export HTTPS_PROXY="http://127.0.0.1:7890"
```

Git config already has `https.proxy=http://127.0.0.1:7890`.
