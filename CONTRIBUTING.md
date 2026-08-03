# Contributing to Dependency Validator

Thank you for helping improve Dependency Validator. Bug fixes, tests,
documentation, and focused feature proposals are welcome.

By participating, you agree to follow the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Before You Start

- Search the [existing issues](https://github.com/Cadeusept/dependency-validator/issues)
  before opening a new one.
- Open an issue before starting a large feature or behavioral change so its
  scope can be agreed upon first.
- Never include access tokens, credentials, private SBOM data, or other secrets
  in issues, commits, fixtures, or logs.

Security vulnerabilities must be reported according to the
[Security Policy](SECURITY.md), not through a public issue.

## Development Setup

Dependency Validator requires Go 1.26.4 or newer.

```bash
git clone https://github.com/Cadeusept/dependency-validator.git
cd dependency-validator
go mod download
go test ./...
go build ./...
```

Create a branch from `master` for your work:

```bash
git switch -c feature/short-description
```

## Making Changes

- Follow standard Go conventions and run `gofmt` on changed Go files.
- Keep changes focused; avoid unrelated refactors in the same pull request.
- Add or update tests when behavior changes.
- Update the README when installation, configuration, output, or supported
  behavior changes.
- Preserve backward compatibility unless a breaking change has been discussed.

Before opening a pull request, run:

```bash
go fmt ./...
go test ./...
go build ./...
golangci-lint run --config=.golangci.yml ./...
```

The project CI uses golangci-lint v2.12.2.

## Pull Requests

Include the following in your pull request:

- A concise description of the problem and solution.
- A link to the related issue, when one exists.
- Tests or a clear explanation of why tests are not required.
- Any compatibility, security, or configuration implications.

Make sure CI passes and respond constructively to review feedback. Maintainers
may request that a large pull request be divided into smaller changes.

## License

Unless you explicitly state otherwise, contributions intentionally submitted
for inclusion in this project are licensed under the
[Apache License 2.0](LICENSE), as described in Section 5 of that license.
