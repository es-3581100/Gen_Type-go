# Contributing to Gen Type-Go

This is an experimental, read-only Phase A publication candidate. Contributions must preserve `PUBLICATION_READINESS ≠ AUTHORITY_RATIFICATION`.

Use Go 1.23 or newer. Run `gofmt`, `go mod verify`, `go test -count=1 ./...`, `go vet ./...`, `go test -race -count=1 ./...`, and `go build ./...` before proposing code. Do not add secrets or private user data. Add tests for malformed or adversarial inputs, stable hashes, no input mutation, and bounds whenever changing parsing or representations. Use synthetic fixtures only.

Propose changes to authority contracts in separate documentation, preserving the immutable historical RFC archives and CV-44 records. Do not wire `internal/phasea`'s pure authorization proof to a request handler, filesystem write, approval workflow, or actual principal identity. No policy publication, root recovery, credential handling, or privileged service is authorized.

Releases require an owner-selected license, independent re-audit of exact bytes, and explicit publication approval. Code reviews should distinguish passing local tests from ratification or production security guarantees.
