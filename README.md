# 📦 Dependency Validator

A simple tool to validate and check whether the versions of specified libraries are up to date.

![GitHub](https://img.shields.io/github/license/Cadeusept/dependency-validator?color=blue)  
![GitHub last commit](https://img.shields.io/github/last-commit/Cadeusept/dependency-validator/working?label=last%20update)  
[![npm](https://img.shields.io/npm/v/dependency-validator?color=green)](https://www.npmjs.com/package/dependency-validator)

---

## 🔍 About

This project helps developers ensure that their dependencies are current, reducing the risk of using outdated or vulnerable packages in your applications. It provides a straightforward way to audit and verify library versions against the latest releases.

Whether you're maintaining a small project or managing a large-scale application, **Dependency Validator** gives you peace of mind by keeping your third-party libraries fresh and secure.

---

## 🚨 Why Use Dependency Validator?

Modern projects drown in dependencies. **Dependency Validator** helps you:
- 🛡️ **Prevent "dependency hell"** by detecting conflicts early.
- 📦 **Remove unused bloat** to speed up installations.
- ⚖️ **Stay license-compliant** with automated checks.
- ⏱️ **Save time** by automating tedious manual audits.

---

## ✨ Features

- ✅ Checks if specified libraries are up to date
- 🔍 Lightweight and easy to integrate
- 🛡️ Helps prevent security issues caused by outdated dependencies

---

## 🚨Dependencies

### To use tool you need anchore/syft to be installed

```bash
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin
```

### And SBoM generated in CycloneDX format:

```bash
syft *your-project-root-path* --output cyclonedx-json=bom.json
```

## 📦 Installation

### Using install script

```bash
curl -sSfL https://raw.githubusercontent.com/Cadeusept/dependency-validator/master/install.sh | sh
```

### Or build manually

```bash
git clone https://github.com/Cadeusept/dependency-validator.git
cd dependency-validator
go build -o dependency-validator ./cmd/main.go
```

---

## 🚀 Usage

To use this validator, create a `.dependency-validator-config.yaml` file listing the repositories you want to monitor. Then run the validator script to compare them with the latest published versions.

> _Note: The validator reads dependency versions from files like `go.mod`, `package.json`, etc., depending on your language._

---

## 📄 Example Config File

Create a `.dependency-validator-config.yaml` at the root of your repository:

```yaml
repos:
  - name: github.com/pedroalbanese/kuznechik
    repo_url: https://github.com/pedroalbanese/kuznechik 

  - name: github.com/stretchr/testify
    repo_url: https://github.com/stretchr/testify

  - name: gitlab.com/private_username/private_repo
    repo_url: https://gitlab.com/private_username/private_repo
    token: your_private_repo_token
```

---

## 📋 Example Output

```
Checking github.com/pedroalbanese/kuznechik...
Up-to-date: v1.2.3

Checking github.com/stretchr/testify...
Outdated: using v1.8.0, latest is v1.9.1

The following dependencies are outdated:
- github.com/stretchr/testify (current: v1.8.0 → latest: v1.9.1)
```

---

## 🗂 Supported Dependency Files

| File              | Ecosystem               | Supported |
|-------------------|-------------------------|-----------|
| `go.mod`          | Go                      | ✅        |
| `package.json`    | JavaScript / Node.js    | ✅        |
| `requirements.txt`| Python                  | ✅        |
| `pyproject.toml`  | Python (PEP 518)        | ✅        |
| `Cargo.toml`      | Rust                    | ✅        |
| `packages.config` | .NET / NuGet            | ✅        |
| `*.csproj`        | .NET / NuGet            | ✅        |
| `Gemfile`         | Ruby                    | ✅        |

---

## 🛠️ CI/CD Integration (GitHub Actions)

You can integrate `dependency-validator` into your CI workflow to automate dependency checks:

```yaml
name: Check Dependencies

on:
  push:
    branches: [main]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install dependency-validator
        run: curl -sSfL https://raw.githubusercontent.com/Cadeusept/dependency-validator/master/install.sh | sh
        
      - name: Install syft (https://github.com/anchore/syft/blob/main/README.md)
        run: curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin
        
      - name: Create SBoM file
        run: syft *your-project-root-path* --output cyclonedx-json=bom.json

      - name: Run validator
        run: dependency-validator
```

---

## 📬 Feedback & Questions

If you have any questions, ideas, or feedback, feel free to [open an issue](https://github.com/Cadeusept/dependency-validator/issues).

---

## ⭐ Support the Project

If you find this tool useful, consider starring the repository to show your support:

⭐ [Star this repo](https://github.com/Cadeusept/dependency-validator)

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
