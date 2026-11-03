# Owner-only first GitHub publication procedure

The verified remote is `https://github.com/es-3581100/Gen_Type-go.git`, configured for `main` and empty at preparation. **Do not use or create the future local checkout** `~/repos/es-3581100/gen_type-go/` as a prerequisite to publication. That directory is intentionally reserved for cloning **after** the first public commit exists.

**Before any commit or push:** MIT was selected by the owner and `LICENSE` is bundled. Confirm copyright attribution. Have an independent reviewer check the **exact** MIT source ZIP recorded in the external publication checkpoint, including its Go tests, inherited race run, security boundaries, and packaging. Obtain explicit owner approval to publish. Phase 1 and all authority-policy activation remain blocked irrespective of this publication.

## Optional temporary owner-controlled publication staging

This creates an isolated disposable checkout, *not* the future local clone. Never overwrite an existing directory, history, or unrelated files. If GitHub is no longer empty, stop and reconcile its history.

```sh
STAGE="$(mktemp -d)"
git clone https://github.com/es-3581100/Gen_Type-go.git "$STAGE/repo"
cd "$STAGE/repo"
git remote -v
git status --short --branch
# Empty GitHub repository should have unborn main and no tracked files.
test "$(git branch --show-current)" = main || {
  echo "Unexpected branch; stop and inspect" >&2; exit 1;
}
test -z "$(git ls-files)" || {
  echo "Repository already has tracked files; stop and inspect" >&2; exit 1;
}
# Supply the independently reviewed ZIP unchanged at this path:
unzip -n "$HOME/Downloads/gen-type-go-initial-publication-candidate-mit-verified.zip"
sha256sum -c SHA256SUMS
export GOWORK=off GOPROXY=off GOSUMDB=off
test -z "$(gofmt -l cmd internal)"
go mod verify
go test -count=1 ./...
go vet ./...
go test -race -count=1 ./...
go build ./...
git status --short
```

**Only after separate explicit owner approval of GitHub publication**, stage exactly the reviewed source tree and inspect the first commit:

```sh
git add README.md AGENTS.md CONTRIBUTING.md SECURITY.md LICENSE \
  .gitignore .github go.mod cmd internal testdata docs tools SHA256SUMS
git diff --cached --check
git diff --cached --stat
git commit -m "feat: publish read-only Gen Type-Go Phase A preview"
git push -u origin main
```

Do **not** run those commit/push commands as part of an unapproved automated review. If local files change after exact-byte review, regenerate the source manifest and repeat the audit before publishing.

## Intended local clone — only after GitHub publication

```sh
REPO="$HOME/repos/es-3581100/gen_type-go"
test ! -e "$REPO" || { echo "Destination exists; stop and inspect" >&2; exit 1; }
mkdir -p "$(dirname "$REPO")"
git clone https://github.com/es-3581100/Gen_Type-go.git "$REPO"
```

No GitHub commit, push, release, policy deployment, or Phase 1 activation was performed during source preparation.
