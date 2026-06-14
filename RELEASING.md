# Releasing keyvind

keyvind is a **multi-module** repository, so a release is a set of git tags —
one per Go module — created from the same commit:

| Module | Import path | Tag form |
| --- | --- | --- |
| core | `github.com/flexphere/keyvind` | `vX.Y.Z` |
| teakit | `github.com/flexphere/keyvind/adapters/teakit` | `adapters/teakit/vX.Y.Z` |
| teakitv2 | `github.com/flexphere/keyvind/adapters/teakitv2` | `adapters/teakitv2/vX.Y.Z` |
| tcellkit | `github.com/flexphere/keyvind/adapters/tcellkit` | `adapters/tcellkit/vX.Y.Z` |

The `examples/*` modules are demos and are **not** released or tagged.

## Versioning policy

[Semantic Versioning](https://semver.org). While `0.y.z`, the API may still
change; bump the **minor** for breaking changes and the **patch** for
backwards-compatible ones. All released modules share a single version number
and are tagged together, so a consumer can rely on matching versions.

Each adapter's `go.mod` pins `require github.com/flexphere/keyvind vX.Y.Z` to the
same version. The `replace github.com/flexphere/keyvind => ../../` directive is
for **local development only** — Go ignores a dependency's `replace`, so external
consumers resolve the core from the pinned tag.

## Cutting a release

1. Update [`CHANGELOG.md`](CHANGELOG.md) (move "Unreleased" to the new version).
2. Bump the pinned core version in every adapter and example `go.mod`
   (`require github.com/flexphere/keyvind` and `.../adapters/*`) to the new
   `vX.Y.Z`.
3. `make all` (and `make ci` for the stricter gate) must be green.
4. Commit, then create and push the tags:

   ```sh
   make tag      VERSION=vX.Y.Z   # tags root + every adapter from HEAD
   make tag-push VERSION=vX.Y.Z   # pushes all the tags to origin
   ```

5. Optionally publish a GitHub release for the root tag:

   ```sh
   gh release create vX.Y.Z --title vX.Y.Z --notes-from-tag
   ```

6. Verify resolution:

   ```sh
   GOFLAGS=-mod=mod go list -m github.com/flexphere/keyvind@vX.Y.Z
   GOFLAGS=-mod=mod go list -m github.com/flexphere/keyvind/adapters/teakit@vX.Y.Z
   ```

`make tag` refuses to run on a dirty tree and validates the `VERSION` is semver.
