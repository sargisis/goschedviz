# How to Create a Release for goschedviz

This project uses GitHub Actions to automatically build and publish binaries when you create a new tag.

## Steps:

### 1. Create a Git Tag
```bash
# Make sure you're on main and up to date
git checkout main
git pull

# Create a new tag (use semantic versioning: v3.0.0, v3.1.0, etc.)
git tag v3.0.0

# Push the tag to GitHub
git push origin v3.0.0
```

### 2. Wait for GitHub Actions
GitHub will automatically:
- Build binaries for Linux (amd64, arm64)
- Build binaries for macOS (Intel, M1/M2)
- Build binaries for Windows (amd64)
- Create a new Release on GitHub
- Attach all binaries to the release

Check progress at: https://github.com/sargisis/goschedviz/actions

---

## Example: First v3 Release

```bash
git tag -a v3.0.0 -m "Initial v3 release: Production-ready TUI & Insights engine"
git push origin v3.0.0
```

After this, users can download with:
```bash
curl -L https://github.com/sargisis/goschedviz/releases/download/v3.0.0/goschedviz-linux-amd64 -o goschedviz
chmod +x goschedviz
```
