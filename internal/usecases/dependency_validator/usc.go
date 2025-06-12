package dependency_validator

import (
	"encoding/json"
	"fmt"
	"github.com/Cadeusept/dependency-validator/internal"
	"golang.org/x/mod/semver"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Cadeusept/dependency-validator/internal/entities"
)

type Usc struct {
	repos        []entities.Repo
	dependencies *entities.SBOM
	outdated     []string
}

func NewUsecase(repos []entities.Repo) *Usc {
	return &Usc{
		repos:        repos,
		outdated:     make([]string, 0),
		dependencies: nil,
	}
}

func (usc *Usc) GetSBOMDependencies() (map[string]entities.DependencyInfo, error) {
	if usc.dependencies == nil {
		return nil, fmt.Errorf("no SBOM loaded")
	}

	dependenciesMap := make(map[string]entities.DependencyInfo)

	for _, component := range usc.dependencies.Components {
		// Skip files and other non-library components
		if component.Type != "library" {
			continue
		}

		dep := entities.DependencyInfo{
			Name:    component.Name,
			Version: component.Version,
			Type:    "library",
		}

		// Extract additional info from properties
		for _, prop := range component.Properties {
			switch prop.Name {
			case "syft:package:type":
				dep.Type = prop.Value
			case "syft:location:0:path":
				dep.Source = prop.Value
			}
		}

		dependenciesMap[dep.Name] = dep
	}

	return dependenciesMap, nil
}

func (usc *Usc) CheckDependencies() []string {
	sbomDeps, err := usc.GetSBOMDependencies()
	if err != nil {
		fmt.Printf("Error getting dependencies from SBoM: %s", err)
	}

	for _, repo := range usc.repos {
		fmt.Printf("Checking %s...\n", repo.Name)

		dependencyInfo, exists := sbomDeps[repo.Name]
		if !exists {
			fmt.Printf("Not found in SBOM. Skipping.\n")
			continue
		}

		latest, err := usc.getLatestGitTag(repo.RepoURL, repo.Token)
		if err != nil {
			fmt.Printf("Failed to get latest version: %v\n", err)
			continue
		}

		if normalizeVersion(dependencyInfo.Version) == normalizeVersion(latest) {
			fmt.Printf("Up-to-date: %s\n", dependencyInfo.Version)
		} else {
			fmt.Printf("%sOutdated: using %s, latest is %s%s\n", internal.TextColorYellow, dependencyInfo.Version, latest, internal.TextColorReset)
			usc.outdated = append(usc.outdated,
				fmt.Sprintf("%s (current: %s → latest: %s)", repo.Name, dependencyInfo.Version, latest))
		}
	}

	return usc.outdated
}

func normalizeVersion(v string) string {
	// Remove 'v' prefix if present
	v = strings.TrimPrefix(v, "v")

	// Split on hyphen and take first part (removes suffixes like -alpha, -beta)
	parts := strings.Split(v, "-")
	if len(parts) > 0 {
		// Only return the first part if it looks like a version (contains digit)
		if strings.ContainsAny(parts[0], "0123456789") {
			return parts[0]
		}
	}
	return v
}

func (usc *Usc) GetAssetVersions() error {
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

func (usc *Usc) ParseSBOM(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read SBOM file: %w", err)
	}

	var sbom entities.SBOM
	if err := json.Unmarshal(data, &sbom); err != nil {
		return fmt.Errorf("failed to parse SBOM: %w", err)
	}

	// Validate it's a CycloneDX SBOM
	if sbom.BomFormat != "CycloneDX" {
		return fmt.Errorf("unsupported SBOM format: %s", sbom.BomFormat)
	}

	usc.dependencies = &sbom

	return nil
}

func (usc *Usc) getLatestGitTag(repoURL, token string) (string, error) {
	if token != "" {
		repoURL = strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", token), 1)
	}
	cmd := exec.Command("git", "ls-remote", "--tags", repoURL)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	var latest string
	for _, line := range lines {
		if strings.Contains(line, "refs/tags/") {
			parts := strings.Split(line, "refs/tags/")
			tag := strings.TrimSpace(parts[1])
			// remove ^{} from annotated tags
			tag = strings.TrimSuffix(tag, "^{}")
			// skip non-semver tags
			if !semver.IsValid(tag) {
				continue
			}
			if latest == "" || semver.Compare(tag, latest) > 0 {
				latest = tag
			}
		}
	}
	if latest == "" {
		return "", fmt.Errorf("no valid semver tags found")
	}
	return latest, nil
}

func (usc *Usc) getLatestNugetVersion(pkg string) (string, error) { //nolint
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

// DetectSBOM scans the directory for common SBoM files and returns the path if found
func (usc *Usc) DetectSBOM(dir string) (string, error) {
	// Common SBoM file names and patterns, lowercase to improve windows compatibility
	sbomPatterns := []string{
		strings.ToLower("sbom.*"),         // sbom.json, sbom.xml, etc.
		strings.ToLower("bom.*"),          // bom.json, bom.xml
		strings.ToLower("*spdx*"),         // spdx.json, spdx.xml
		strings.ToLower("*cyclonedx*"),    // cyclonedx.json, cyclonedx.xml
		strings.ToLower(".syft*"),         // .syft.json (anchore/syft default output)
		strings.ToLower("*bom*.json"),     // any JSON file with "bom" in name
		strings.ToLower("*bom*.xml"),      // any XML file with "bom" in name
		strings.ToLower("*inventory*"),    // inventory files
		strings.ToLower("*dependencies*"), // dependencies files
		strings.ToLower("*components*"),   // components files
	}

	// Common SBoM file extensions
	sbomExtensions := []string{
		".json",
		".xml",
		".spdx",
		".cdx",
	}

	// Walk through the directory looking for matching files
	var foundFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		filename := strings.ToLower(info.Name())
		ext := strings.ToLower(filepath.Ext(filename))

		// Check if extension is one of our SBoM extensions
		hasValidExt := false
		for _, sbomExt := range sbomExtensions {
			if ext == sbomExt {
				hasValidExt = true
				break
			}
		}

		if !hasValidExt {
			return nil
		}

		// Check if filename matches any of our patterns
		for _, pattern := range sbomPatterns {
			matched, _ := filepath.Match(pattern, filename)
			if matched {
				foundFiles = append(foundFiles, path)
				break
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error scanning directory for SBoM: %w", err)
	}

	// Prioritize certain files if multiple are found
	if len(foundFiles) > 0 {
		// Check for exact matches first
		for _, file := range foundFiles {
			base := strings.ToLower(filepath.Base(file))
			if base == "sbom.json" || base == "bom.json" {
				return file, nil
			}
		}

		// Check for syft output
		for _, file := range foundFiles {
			if strings.Contains(strings.ToLower(filepath.Base(file)), ".syft") {
				return file, nil
			}
		}

		// Return the first found file
		return foundFiles[0], nil
	}

	return "", fmt.Errorf("no SBoM file found in directory")
}
