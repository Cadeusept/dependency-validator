package dependency_validator

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Cadeusept/dependency-validator/internal/entities"
)

type Usc struct {
	repos         []entities.Repo
	dependencies  []string
	assetVersions map[string]string
	outdated      []string
}

func NewUsecase(repos []entities.Repo) *Usc {
	return &Usc{
		repos:         repos,
		outdated:      make([]string, 0),
		assetVersions: make(map[string]string),
		dependencies:  make([]string, 0),
	}
}

func (usc Usc) GetAssetVersions() error {
	path := "obj/project.assets.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var parsed struct {
		Libraries map[string]interface{} `json:"libraries"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	versions := make(map[string]string)
	for full := range parsed.Libraries {
		parts := strings.Split(full, "/")
		if len(parts) == 2 {
			versions[parts[0]] = parts[1]
		}
	}
	return nil
}

func (usc Usc) DetectDependencyFile() (string, error) {
	files := []string{
		"go.mod", "package.json", "requirements.txt",
		"pyproject.toml", "Gemfile", "Cargo.toml", "packages.config",
	}
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			return f, nil
		}
	}
	// Look for *.csproj
	matches, _ := filepath.Glob("*.csproj")
	if len(matches) > 0 {
		return matches[0], nil
	}
	return "", fmt.Errorf("no known dependency file found")
}

func (usc Usc) CheckDependencies() []string {
	used := make(map[string]bool)
	for _, dep := range usc.dependencies {
		used[dep] = true
	}

	for _, repo := range usc.repos {
		fmt.Printf("Checking %s...\n", repo.Name)

		if !used[repo.Name] {
			fmt.Printf("Not used in this project. Skipping.\n")
			continue
		}

		latest, err := usc.getLatestGitTag(repo.RepoURL, repo.Token)
		if err != nil {
			fmt.Printf("Failed to get latest version: %v\n", err)
			continue
		}

		currentVersion, err := usc.getCurrentVersion(repo.Name, usc.assetVersions)
		if err != nil {
			fmt.Printf("Could not determine current version: %v\n", err)
			continue
		}

		if strings.TrimPrefix(currentVersion, "v") == strings.TrimPrefix(latest, "v") {
			fmt.Printf("Up-to-date: %s\n", currentVersion)
		} else {
			fmt.Printf("Outdated: using %s, latest is %s\n", currentVersion, latest)
			usc.outdated = append(usc.outdated, fmt.Sprintf("%s (current: %s → latest: %s)", repo.Name, currentVersion, latest))
		}
	}

	// NuGet check
	for _, dep := range usc.dependencies {
		if usc.usedInConfig(dep, usc.repos) {
			continue
		}

		currentVersion, err := usc.getCurrentVersion(dep, usc.assetVersions)
		if err != nil {
			continue
		}

		latest, err := usc.getLatestNugetVersion(dep)
		if err != nil {
			continue
		}

		if currentVersion != latest {
			fmt.Printf("NuGet outdated: %s (current: %s → latest: %s)\n", dep, currentVersion, latest)
			usc.outdated = append(usc.outdated, fmt.Sprintf("%s (current: %s → latest: %s)", dep, currentVersion, latest))
		}
	}

	return usc.outdated
}

func (usc Usc) ParseDependencies(file string) error {
	var err error
	switch filepath.Base(file) {
	case "go.mod":
		usc.dependencies, err = usc.parseTextLines("go.mod")
		return err
	case "package.json":
		usc.dependencies, err = usc.parseJSONDependencies("package.json")
		return err
	case "requirements.txt":
		usc.dependencies, err = usc.parseTextLines("requirements.txt")
		return err
	case "pyproject.toml", "Cargo.toml":
		usc.dependencies, err = usc.parseTextLines(file)
		return err
	case "Gemfile":
		usc.dependencies, err = usc.parseTextLines("Gemfile")
		return err
	case "packages.config":
		usc.dependencies, err = usc.parsePackagesConfig(file)
		return err
	default:
		if strings.HasSuffix(file, ".csproj") {
			usc.dependencies, err = usc.parseCSPROJ(file)
			return err
		}
	}
	return fmt.Errorf("unsupported dependency file")
}

func (usc Usc) parseTextLines(file string) ([]string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var deps []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			deps = append(deps, strings.Fields(line)[0])
		}
	}
	return deps, nil
}

func (usc Usc) parseJSONDependencies(file string) ([]string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	deps := []string{}
	if depsMap, ok := parsed["dependencies"].(map[string]interface{}); ok {
		for k := range depsMap {
			deps = append(deps, k)
		}
	}
	return deps, nil
}

func (usc Usc) parsePackagesConfig(file string) ([]string, error) {
	type Package struct {
		ID string `xml:"id,attr"`
	}
	type Packages struct {
		Packages []Package `xml:"package"`
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var parsed Packages
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	var deps []string
	for _, p := range parsed.Packages {
		deps = append(deps, p.ID)
	}
	return deps, nil
}

func (usc Usc) parseCSPROJ(file string) ([]string, error) {
	type PackageReference struct {
		Include string `xml:"Include,attr"`
	}
	type Project struct {
		ItemGroups []struct {
			PackageReferences []PackageReference `xml:"PackageReference"`
		} `xml:"ItemGroup"`
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var proj Project
	if err := xml.Unmarshal(data, &proj); err != nil {
		return nil, err
	}
	var deps []string
	for _, group := range proj.ItemGroups {
		for _, pr := range group.PackageReferences {
			deps = append(deps, pr.Include)
		}
	}
	return deps, nil
}

func (usc Usc) getLatestGitTag(repoURL, token string) (string, error) {
	args := []string{"ls-remote", "--tags", repoURL}
	cmd := exec.Command("git", args...)
	if token != "" {
		repoURL = strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", token), 1)
		cmd = exec.Command("git", "ls-remote", "--tags", repoURL)
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	var lastTag string
	for _, line := range lines {
		if strings.Contains(line, "refs/tags/") {
			parts := strings.Split(line, "refs/tags/")
			lastTag = strings.TrimSpace(parts[1])
		}
	}
	return lastTag, nil
}

func (usc Usc) getCurrentVersion(dep string, assets map[string]string) (string, error) {
	if _, err := os.Stat("go.mod"); err == nil {
		cmd := exec.Command("go", "list", "-m", "all")
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, dep+" ") {
				return strings.Fields(line)[1], nil
			}
		}
	}

	if v, ok := assets[dep]; ok {
		return v, nil
	}

	return "", fmt.Errorf("version for %s not found", dep)
}

func (usc Usc) usedInConfig(dep string, repos []entities.Repo) bool {
	for _, r := range repos {
		if r.Name == dep {
			return true
		}
	}
	return false
}

func (usc Usc) getLatestNugetVersion(pkg string) (string, error) {
	url := fmt.Sprintf("https://api.nuget.org/v3-flatcontainer/%s/index.json", strings.ToLower(pkg))
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("NuGet package %s not found", pkg)
	}
	var result struct {
		Versions []string `json:"versions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Versions) == 0 {
		return "", fmt.Errorf("no versions found for %s", pkg)
	}
	return result.Versions[len(result.Versions)-1], nil
}
