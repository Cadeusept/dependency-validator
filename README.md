<div align="center">
  <h1>Dependency Validator</h1>
  <p><strong>Keep CycloneDX dependencies aligned with the latest semantic Git tags.</strong></p>

  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.26.4-00ADD8?logo=go&amp;logoColor=white" alt="Go 1.26.4"></a>
  <a href="https://cyclonedx.org/"><img src="https://img.shields.io/badge/SBOM-CycloneDX-6A4C93?logo=owasp&amp;logoColor=white" alt="CycloneDX SBOM"></a>
  <a href="https://github.com/anchore/syft"><img src="https://img.shields.io/badge/Generated_with-Syft-623CE4?logo=github&amp;logoColor=white" alt="Generated with Syft"></a>
  <a href="https://git-scm.com/"><img src="https://img.shields.io/badge/Versions-Git_tags-F05032?logo=git&amp;logoColor=white" alt="Git tag versions"></a>
  <a href="https://github.com/Cadeusept"><img src="https://img.shields.io/badge/Author-Cadeusept-8B9AFF?logo=github&amp;logoColor=white" alt="Author: Cadeusept"></a>
</div>

<div align="center">
  <a href="https://github.com/Cadeusept/dependency-validator"><img src="https://img.shields.io/github/repo-size/Cadeusept/dependency-validator" alt="Repository size"></a>
  <a href="https://github.com/Cadeusept/dependency-validator/commits/master"><img src="https://img.shields.io/github/last-commit/Cadeusept/dependency-validator/master" alt="Last commit"></a>
  <a href="https://github.com/Cadeusept/dependency-validator/commits/master"><img src="https://img.shields.io/github/commit-activity/m/Cadeusept/dependency-validator" alt="Monthly commit activity"></a>
  <a href="https://github.com/Cadeusept/dependency-validator/pulls"><img src="https://img.shields.io/github/issues-pr/Cadeusept/dependency-validator" alt="Open pull requests"></a>
  <a href="https://github.com/Cadeusept/dependency-validator/graphs/contributors"><img src="https://img.shields.io/github/contributors/Cadeusept/dependency-validator" alt="Contributors"></a>
</div>

<div align="center">
  <a href="https://github.com/Cadeusept/dependency-validator/actions/workflows/go.yml"><img src="https://github.com/Cadeusept/dependency-validator/actions/workflows/go.yml/badge.svg?branch=master" alt="Go CI"></a>
  <a href="https://pkg.go.dev/github.com/Cadeusept/dependency-validator"><img src="https://pkg.go.dev/badge/github.com/Cadeusept/dependency-validator.svg" alt="Go Reference"></a>
  <a href="https://github.com/Cadeusept/dependency-validator/releases/latest"><img src="https://img.shields.io/github/v/release/Cadeusept/dependency-validator?sort=semver" alt="Latest release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-D22128?logo=apache&amp;logoColor=white" alt="Apache 2.0 license"></a>
</div>

Dependency Validator is a command-line utility that compares dependencies in a
[CycloneDX](https://cyclonedx.org/) software bill of materials (SBOM) with the
latest semantic-version tags in their source repositories.

It is useful in local development and CI when you want an explicit allowlist of
dependencies that must stay on the newest tagged release.

## How it works

1. Dependency Validator finds an SBOM in the current directory.
2. It reads library names and versions from a CycloneDX JSON document.
3. For every repository in `.dependency-validator-config.yaml`, it finds the
   highest valid semantic-version Git tag.
4. It exits with status `1` when at least one configured dependency is outdated.

Only dependencies listed in the configuration are checked. Components that are
present in the SBOM but absent from the configuration are ignored.

## Requirements

- Git, used to read remote repository tags.
- A CycloneDX JSON SBOM. [Syft](https://github.com/anchore/syft) is the
  recommended generator.
- Go 1.26.4 or newer when installing or building from source.

## Installation

Install the latest tagged version with Go:

```bash
go install github.com/Cadeusept/dependency-validator@latest
```

Prebuilt archives are available on the
[GitHub Releases page](https://github.com/Cadeusept/dependency-validator/releases).

To build the current source manually:

```bash
git clone https://github.com/Cadeusept/dependency-validator.git
cd dependency-validator
go build -o dependency-validator .
```

## Configuration

Create `.dependency-validator-config.yaml` in the project directory:

```yaml
repos:
  - name: github.com/pedroalbanese/kuznechik
    repo_url: https://github.com/pedroalbanese/kuznechik

  - name: github.com/stretchr/testify
    repo_url: https://github.com/stretchr/testify
    token: ${GITHUB_TOKEN}
```

`name` must match the component name in the SBOM. `repo_url` must be a Git
repository whose releases use valid semantic-version tags such as `v1.2.3`.
`token` is optional, and environment variables in the configuration are
expanded before it is parsed.

Do not commit access tokens. Pass them through an environment variable or your
CI secret store.

## Usage

Install Syft by following its
[official installation instructions](https://github.com/anchore/syft#installation),
then generate a CycloneDX JSON SBOM:

```bash
syft . --output cyclonedx-json=bom.json
```

Run the validator from the directory containing the SBOM and configuration:

```bash
dependency-validator
```

Example output:

```text
Found SBoM file: bom.json
Checking github.com/stretchr/testify...
Outdated: using v1.8.0, latest is v1.9.1

The following dependencies are outdated:
 - github.com/stretchr/testify (current: v1.8.0 → latest: v1.9.1)
```

## GitHub Actions example

Keep `.dependency-validator-config.yaml` in your repository and add a workflow
step like this:

```yaml
name: Check dependencies

on:
  pull_request:
  push:
    branches: [master]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.26.4'

      - name: Install Dependency Validator
        run: go install github.com/Cadeusept/dependency-validator@latest

      - name: Install Syft
        run: curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b ./bin

      - name: Generate CycloneDX SBOM
        run: ./bin/syft . --output cyclonedx-json=bom.json

      - name: Check dependency versions
        run: "$(go env GOPATH)/bin/dependency-validator"
```

## Scope and limitations

- Only CycloneDX JSON input is parsed.
- Upstream versions must be valid semantic-version Git tags.
- The utility checks whether configured versions are current; it does not scan
  for vulnerabilities, license violations, dependency conflicts, or unused
  packages.
- Repository access requires network connectivity and may require a token for
  private repositories or rate-limited environments.

## Contributing

Bug reports and pull requests are welcome. Read the
[Contributing Guide](CONTRIBUTING.md) before opening a pull request, and follow
the [Code of Conduct](CODE_OF_CONDUCT.md) in all project spaces.

## Security

Please report vulnerabilities privately according to the
[Security Policy](SECURITY.md). Do not disclose security-sensitive details in a
public issue.

## License

Dependency Validator is available under the
[Apache License 2.0](LICENSE). You may use, modify, and redistribute it under
the license terms. Redistributions must preserve the applicable copyright and
attribution notices in [NOTICE](NOTICE).
