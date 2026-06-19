# Build and deploy

Internal notes for maintaining **ai-cli** releases and the Homebrew tap.

---

## Prerequisites

| Tool | Purpose |
|------|---------|
| Go 1.25+ | Local builds |
| [GoReleaser v2.10+](https://goreleaser.com/) | Release artifacts + Homebrew cask |
| `gh` CLI | Optional: create repo, watch Actions |
| GitHub repo | `karpulix/ai-cli` |
| Homebrew tap | [karpulix/homebrew-tools](https://github.com/karpulix/homebrew-tools) |

---

## Local development

```bash
go mod tidy
go build -o ai-cli .
./ai-cli --version          # ai-cli vdev
```

Run TUI directly:

```bash
./ai-cli
```

### Build with embedded version (manual)

```bash
go build -ldflags "\
  -X github.com/karpulix/ai-cli/internal/version.Version=0.1.0 \
  -X github.com/karpulix/ai-cli/internal/version.Commit=$(git rev-parse --short HEAD) \
  -X github.com/karpulix/ai-cli/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o ai-cli .
```

Module path in `go.mod` is `github.com/karpulix/ai-cli` — ldflags and imports must match until renamed.

Version variables live in `internal/version/version.go`:

| Variable | Release value | Local default |
|----------|---------------|---------------|
| `Version` | `0.1.0` (from tag, no `v`) | `dev` |
| `Commit` | short SHA | `none` |
| `Date` | build timestamp | `unknown` |

---

## GoReleaser snapshot (no publish)

Dry-run release into `dist/`:

```bash
goreleaser release --snapshot --clean
ls dist/
```

Config: `.goreleaser.yaml`

Build targets:

| GOOS | GOARCH |
|------|--------|
| darwin | amd64, arm64 |
| linux | amd64, arm64 |
| windows | amd64, arm64 |

Artifacts: `ai-cli_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows) + `checksums.txt`.

---

## GitHub release flow

### One-time setup

1. Push code to `github.com/karpulix/ai-cli`.
2. Ensure workflow exists: `.github/workflows/release.yml`.
3. Add repository secret:

   | Secret | Scope | Used for |
   |--------|-------|----------|
   | `HOMEBREW_TAP_GITHUB_TOKEN` | Fine-grained PAT, **Contents: Read and write** on `homebrew-tools` | Push `Casks/ai-cli.rb` to tap |

   `GITHUB_TOKEN` is provided automatically by Actions (`contents: write` in workflow).

4. Tap repo `karpulix/homebrew-tools` must exist (formulas at repo root, same layout as `macosloginwatcher.rb`).

### Every release

```bash
# 1. Commit and push main
git add .
git commit -m "..."
git push origin main

# 2. Tag and push
git tag v0.1.0
git push origin v0.1.0
```

### What happens automatically

1. GitHub Action **Release** runs on tag `v*`.
2. GoReleaser:
   - cross-compiles all targets with ldflags
   - creates GitHub Release `v0.1.0`
   - uploads archives + checksums
   - commits `Casks/ai-cli.rb` to `karpulix/homebrew-tools`

### Verify

```bash
gh run list --workflow=Release
gh release view v0.1.0
```

Check tap:

```bash
git clone https://github.com/karpulix/homebrew-tools /tmp/homebrew-tools
cat /tmp/homebrew-tools/Casks/ai-cli.rb
```

Install from tap:

```bash
brew install karpulix/tools/ai-cli
ai-cli --version
```

---

## Fixing a bad release

GoReleaser will not overwrite an existing release for the same tag.

```bash
# Delete remote tag (only if safe — no users on broken build)
git push origin :refs/tags/v0.1.0
git tag -d v0.1.0

# Fix code, tag patch version
git tag v0.1.1
git push origin v0.1.1
```

Delete the broken GitHub Release manually in the UI if needed.

---

## Homebrew tap config

In `.goreleaser.yaml` — `homebrew_casks` (not deprecated `brews`). Cask скачивает готовый бинарник; на Linux не нужен gcc/clang.

One-time migration in `karpulix/homebrew-tools`:

1. Delete old `ai-cli.rb` (formula at repo root).
2. Add `tap_migrations.json`:

```json
{
  "ai-cli": "ai-cli"
}
```

User install command (Homebrew 5.0.6+):

```bash
brew install karpulix/tools/ai-cli
```

After brew install, shell widget still needs one-time setup:

```bash
ai-cli install zsh
source ~/.zshrc
```

---

## CI secrets checklist

| Item | Where |
|------|-------|
| `HOMEBREW_TAP_GITHUB_TOKEN` | `karpulix/ai-cli` → Settings → Secrets |
| PAT owner | Account `karpulix` with push access to `homebrew-tools` |
| Workflow permissions | `contents: write` (already in workflow) |

---

## Optional: rename module to karpulix

Currently `go.mod` uses `github.com/karpulix/ai-cli`. To align with GitHub org/user:

1. Change `module` in `go.mod`.
2. Replace import paths across the repo.
3. Update ldflags in `.goreleaser.yaml`.
4. Tag a new release.

Not required for releases — cosmetic consistency only.

---

## Project layout (release-related)

```
.github/workflows/release.yml   # CI trigger on tags
.goreleaser.yaml                # builds, archives, homebrew cask
internal/version/version.go     # version vars (ldflags target)
main.go                         # --version flag
internal/app/info.go            # version in TUI Info panel
```
