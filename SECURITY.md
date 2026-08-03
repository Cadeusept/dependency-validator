# Security Policy

## Supported Versions

Security fixes are applied to the latest stable `v1.x` release and the current
`master` branch. Older releases may not receive fixes.

| Version | Supported |
| --- | --- |
| Latest `v1.x` release | Yes |
| `master` | Yes, development version |
| Older releases | No |

Users should upgrade to the latest stable release whenever possible.

## Reporting a Vulnerability

Do not disclose suspected vulnerabilities in a public issue, discussion, pull
request, or commit.

Use GitHub's private
[Report a vulnerability](https://github.com/Cadeusept/dependency-validator/security/advisories/new)
form. Include:

- The affected version or commit.
- A description of the vulnerability and its potential impact.
- Minimal reproduction steps or a proof of concept.
- Any suggested remediation, if known.
- Whether and when you intend to disclose the issue publicly.

Remove access tokens, credentials, proprietary SBOM contents, and unrelated
personal data from the report.

The maintainer will acknowledge the report as soon as practical, assess its
impact, coordinate a fix and release, and credit the reporter if requested.
Please allow time for a patch to be released before public disclosure.

## Security Scope

Reports are especially useful for issues involving:

- Exposure of repository tokens or credentials.
- Command or argument injection through repository URLs, SBOM contents, or
  configuration values.
- Unsafe parsing of untrusted SBOM files.
- Incorrect handling of private repository authentication.
- Release artifact or dependency supply-chain integrity.

General bugs, feature requests, and outdated dependency reports that do not
create a security risk should use
[GitHub Issues](https://github.com/Cadeusept/dependency-validator/issues).

## Disclosure

Validated vulnerabilities will be addressed privately. After a fix is
available, the maintainer may publish a GitHub Security Advisory describing the
affected versions, impact, remediation, and reporter credit.
