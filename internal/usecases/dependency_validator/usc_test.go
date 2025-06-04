package dependency_validator

import (
	"os"
	"testing"

	"github.com/Cadeusept/dependency-validator/internal/entities"
	"github.com/stretchr/testify/assert"
)

// Helper to write temp files
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	fullPath := tmpDir + "/" + name
	err := os.WriteFile(fullPath, []byte(content), 0644)
	assert.NoError(t, err)
	return fullPath
}

func TestParseTextLines(t *testing.T) {
	content := `
# This is a comment
github.com/example/lib v1.2.3
  another/repo   v2.0.0
`
	path := writeTempFile(t, "deps.txt", content)

	validator := Usc{}
	deps, err := validator.parseTextLines(path)

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"github.com/example/lib",
		"another/repo",
	}, deps)
}

func TestParseJSONDependencies(t *testing.T) {
	content := `{
		"dependencies": {
			"express": "^4.17.1",
			"axios": "^0.24.0"
		}
	}`
	path := writeTempFile(t, "package.json", content)

	validator := Usc{}
	deps, err := validator.parseJSONDependencies(path)

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"express", "axios"}, deps)
}

func TestParsePackagesConfig(t *testing.T) {
	content := `
<packages>
	<package id="Newtonsoft.Json" version="13.0.1" />
	<package id="Serilog" version="2.10.0" />
</packages>
`
	path := writeTempFile(t, "packages.config", content)

	validator := Usc{}
	deps, err := validator.parsePackagesConfig(path)

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"Newtonsoft.Json", "Serilog"}, deps)
}

func TestParseCSPROJ(t *testing.T) {
	content := `
<Project Sdk="Microsoft.NET.Sdk">
  <ItemGroup>
    <PackageReference Include="NUnit" Version="3.13.2" />
    <PackageReference Include="Moq" Version="4.16.1" />
  </ItemGroup>
</Project>`
	path := writeTempFile(t, "project.csproj", content)

	validator := Usc{}
	deps, err := validator.parseCSPROJ(path)

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"NUnit", "Moq"}, deps)
}

func TestUsedInConfig(t *testing.T) {
	validator := Usc{}
	repos := []entities.Repo{
		{Name: "github.com/example/lib"},
		{Name: "other/repo"},
	}

	assert.True(t, validator.usedInConfig("github.com/example/lib", repos))
	assert.False(t, validator.usedInConfig("not/found", repos))
}

func TestGetLatestNugetVersion_RealNuGet(t *testing.T) {
	validator := Usc{}

	// Example: Newtonsoft.Json is a very common NuGet package
	version, err := validator.getLatestNugetVersion("Newtonsoft.Json")

	assert.NoError(t, err)
	assert.Regexp(t, `^\d+\.\d+\.\d+`, version, "should return a semantic version")
}

func TestGetCurrentVersion_FromAssets(t *testing.T) {
	validator := Usc{}
	assets := map[string]string{
		"github.com/example/lib": "v1.2.3",
	}
	version, err := validator.getCurrentVersion("github.com/example/lib", assets)
	assert.NoError(t, err)
	assert.Equal(t, "v1.2.3", version)
}

func TestGetCurrentVersion_NotFound(t *testing.T) {
	validator := Usc{}
	_, err := validator.getCurrentVersion("nonexistent/pkg", map[string]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version for nonexistent/pkg not found")
}
