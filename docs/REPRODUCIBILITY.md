# Reproducible build and source packaging

Environment used for preparation: Linux/amd64; Go 1.23.2; Python 3.13.5; Git 2.47.3. Go module uses the standard library only and has no `go.sum` because there are no third-party module dependencies. The `GOWORK=off GOPROXY=off GOSUMDB=off` settings prevent workspace substitution or module downloads.

```sh
export GOWORK=off GOPROXY=off GOSUMDB=off
test -z "$(gofmt -l cmd internal)"
go mod verify
go test -count=1 ./...
go vet ./...
go test -race -count=1 ./...
go build ./...
go run ./cmd/gen-type-go validate testdata/tiny_graph.json
go run ./cmd/gen-type-go roundtrip testdata/tiny_graph.json
go run ./cmd/gen-type-go sha256 testdata/tiny_graph.json
```

To reproduce byte-identical source packaging, run from the project directory:

```sh
python3 tools/package_source.py /tmp/gen-type-go-initial-publication-candidate.zip
sha256sum /tmp/gen-type-go-initial-publication-candidate.zip
```

The packager uses sorted paths, a fixed ZIP timestamp, normalized permissions, and a SHA256SUMS file *inside* the ZIP indexing all source entries. It ignores ZIP outputs and preexisting SHA256SUMS when re-packaging an extracted copy. The outer source ZIP digest belongs in the **external publication checkpoint** to avoid a circular self-hash.

CI pins GitHub Actions by full commit SHA, sets read-only token permission and disables external Go module resolution. CI itself has not yet run on GitHub; local execution is separate evidence. Cross-build successes do not establish target-platform runtime behavior.
