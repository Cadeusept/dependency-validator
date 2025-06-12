package dependency_validator

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Cadeusept/dependency-validator/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures
const (
	testCycloneDXSBOM = `{
		"bomFormat": "CycloneDX",
		"specVersion": "1.6",
		"components": [
			{
			  "bom-ref": "pkg:golang/github.com/stretchr/testify@v1.9.0?package-id=1e92eaf826745a46",
			  "type": "library",
			  "name": "github.com/stretchr/testify",
			  "version": "v1.9.0",
			  "cpe": "cpe:2.3:a:stretchr:testify:v1.9.0:*:*:*:*:*:*:*",
			  "purl": "pkg:golang/github.com/stretchr/testify@v1.9.0",
			  "properties": [
				{
				  "name": "syft:package:foundBy",
				  "value": "go-module-file-cataloger"
				},
				{
				  "name": "syft:package:language",
				  "value": "go"
				},
				{
				  "name": "syft:package:type",
				  "value": "go-module"
				},
				{
				  "name": "syft:package:metadataType",
				  "value": "go-module-entry"
				},
				{
				  "name": "syft:location:0:path",
				  "value": "/go.mod"
				},
				{
				  "name": "syft:metadata:h1Digest",
				  "value": "h1:HtqpIVDClZ4nwg75+f6Lvsy/wHu+3BoSGCbBAcpTsTg="
				}
			  ]
			},
			{
			  "bom-ref": "pkg:golang/golang.org/x/mod@v0.24.0?package-id=0ab7d18217876fde",
			  "type": "library",
			  "name": "golang.org/x/mod",
			  "version": "v0.24.0",
			  "cpe": "cpe:2.3:a:golang:x\\/mod:v0.24.0:*:*:*:*:*:*:*",
			  "purl": "pkg:golang/golang.org/x/mod@v0.24.0",
			  "properties": [
				{
				  "name": "syft:package:foundBy",
				  "value": "go-module-file-cataloger"
				},
				{
				  "name": "syft:package:language",
				  "value": "go"
				},
				{
				  "name": "syft:package:type",
				  "value": "go-module"
				},
				{
				  "name": "syft:package:metadataType",
				  "value": "go-module-entry"
				},
				{
				  "name": "syft:location:0:path",
				  "value": "/go.mod"
				},
				{
				  "name": "syft:metadata:h1Digest",
				  "value": "h1:ZfthKaKaT4NrhGVZHO1/WDTwGES4De8KtWO0SIbNJMU="
				}
			  ]
			},
			{
				"type": "file",
				"name": "/go.mod"
			}
		],
		"metadata": {
			"timestamp": "2025-06-12T15:05:41+03:00",
			"tools": {
				"components": [
					{
						"type": "application",
						"name": "syft",
						"version": "1.27.0"
					}
				]
			}
		}
	}`

	gitTagsOutput = `5bdb3a5a3c3325a81f7a14539a644e5d5b1a9f0a	refs/tags/v1.2.3
8d7b41f2a4e6b5f0c3e9d7a6b5c4d3e2f1a0b9e	refs/tags/v2.0.0^{}
`
)

func TestNewUsecase(t *testing.T) {
	t.Parallel()

	repos := []entities.Repo{
		{Name: "test-repo", RepoURL: "https://github.com/test/repo.git"},
	}

	usc := NewUsecase(repos)
	assert.NotNil(t, usc)
	assert.Equal(t, repos, usc.repos)
	assert.NotNil(t, usc.outdated)
	assert.Nil(t, usc.dependencies)
}

func TestParseSBOM(t *testing.T) {
	t.Parallel()

	// Create temp SBOM file
	tmpFile, err := os.CreateTemp("", "sbom-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(testCycloneDXSBOM)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	usc := NewUsecase(nil)
	err = usc.ParseSBOM(tmpFile.Name())
	require.NoError(t, err)

	assert.NotNil(t, usc.dependencies)
	assert.Equal(t, "CycloneDX", usc.dependencies.BomFormat)
	assert.Len(t, usc.dependencies.Components, 3)

	// Test invalid SBOM format
	invalidSBOM := `{"bomFormat": "SPDX", "specVersion": "2.2"}`
	err = os.WriteFile(tmpFile.Name(), []byte(invalidSBOM), 0644)
	require.NoError(t, err)

	err = usc.ParseSBOM(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported SBOM format")
}

func TestGetSBOMDependencies(t *testing.T) {
	t.Parallel()

	usc := NewUsecase(nil)
	usc.dependencies = &entities.SBOM{
		Components: []entities.Component{
			{
				Type:    "library",
				Name:    "github.com/example/repo1",
				Version: "v1.2.3",
				Properties: []entities.ComponentProperty{
					{Name: "syft:package:type", Value: "go-module"},
					{Name: "syft:location:0:path", Value: "/go.mod"},
				},
			},
			{
				Type: "file",
				Name: "/go.mod",
			},
		},
	}

	deps, err := usc.GetSBOMDependencies()
	require.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Equal(t, "v1.2.3", deps["github.com/example/repo1"].Version)
	assert.Equal(t, "go-module", deps["github.com/example/repo1"].Type)
	assert.Equal(t, "/go.mod", deps["github.com/example/repo1"].Source)

	// Test no SBOM loaded
	uscNoSBOM := NewUsecase(nil)
	_, err = uscNoSBOM.GetSBOMDependencies()
	assert.Error(t, err)
}

func TestCheckDependencies(t *testing.T) {
	t.Parallel()

	// Mock git command
	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("echo", gitTagsOutput)
	}
	defer func() { execCommand = exec.Command }()

	// Setup test case
	repos := []entities.Repo{
		{
			Name:    "github.com/stretchr/testify",
			RepoURL: "https://github.com/stretchr/testify",
		},
	}

	usc := NewUsecase(repos)
	usc.dependencies = &entities.SBOM{
		Components: []entities.Component{
			{
				Type:    "library",
				Name:    "github.com/stretchr/testify",
				Version: "v1.9.0",
			},
			{
				Type:    "library",
				Name:    "golang.org/x/mod",
				Version: "v0.24.0",
			},
		},
	}

	outdated := usc.CheckDependencies()
	assert.Len(t, outdated, 1)
	assert.Contains(t, outdated[0], "github.com/stretchr/testify")
}

//func TestGetLatestNugetVersion(t *testing.T) {
//	t.Parallel()
//
//	// Setup test server
//	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		if strings.Contains(r.URL.Path, "microsoft.extensions.logging") {
//			_, _ = io.WriteString(w, `{"versions": ["5.0.0", "6.0.0"]}`)
//		} else {
//			w.WriteHeader(http.StatusNotFound)
//		}
//	}))
//	defer ts.Close()
//
//	// Override NuGet URL for testing
//	oldURL := dependency_validator.NuGetIndexURL
//	dependency_validator.NuGetIndexURL = ts.URL + "/v3-flatcontainer/%s/index.json"
//	defer func() { dependency_validator.NuGetIndexURL = oldURL }()
//
//	usc := dependency_validator.NewUsecase(nil)
//
//	// Test happy path
//	version, err := usc.GetLatestNugetVersion("Microsoft.Extensions.Logging")
//	require.NoError(t, err)
//	assert.Equal(t, "6.0.0", version)
//
//	// Test not found
//	_, err = usc.GetLatestNugetVersion("Nonexistent.Package")
//	assert.Error(t, err)
//}

func TestDetectSBOM(t *testing.T) {
	t.Parallel()

	// Create temp dir with test files
	tmpDir, err := os.MkdirTemp("", "sbom-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	files := []struct {
		name    string
		content string
	}{
		{"bom.json", testCycloneDXSBOM},
		{"other.txt", "not an SBOM"},
		{"spdx.json", `{"SPDXID": "SPDXRef-DOCUMENT"}`},
	}

	for _, file := range files {
		err = os.WriteFile(filepath.Join(tmpDir, file.name), []byte(file.content), 0644)
		require.NoError(t, err)
	}

	usc := NewUsecase(nil)

	// Test detection finds bom.json first
	path, err := usc.DetectSBOM(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, "bom.json"), path)

	// Remove bom.json and test fallback to spdx.json
	err = os.Remove(filepath.Join(tmpDir, "bom.json"))
	require.NoError(t, err)

	path, err = usc.DetectSBOM(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, "spdx.json"), path)

	// Test no SBOM found
	emptyDir, err := os.MkdirTemp("", "empty")
	require.NoError(t, err)
	defer os.RemoveAll(emptyDir)

	_, err = usc.DetectSBOM(emptyDir)
	assert.Error(t, err)
}

func TestNormalizeVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"v1.2.3", "1.2.3"},
		{"2.0.0-alpha", "2.0.0"},
		{"3.0.0", "3.0.0"},
		{"release-4.5.6", "release-4.5.6"}, // Preserve full string since "release" doesn't contain numbers
		{"4.5.6-release", "4.5.6"},         // Still strips suffix
		{"stable-5.0", "stable-5.0"},       // Preserve full string
		{"5.0-stable", "5.0"},              // Strips suffix
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeVersion(tt.input))
		})
	}
}

// TestErrorPaths tests error cases
func TestErrorPaths(t *testing.T) {
	t.Parallel()

	// Test ParseSBOM with invalid file
	usc := NewUsecase(nil)
	err := usc.ParseSBOM("nonexistent.json")
	assert.Error(t, err)

	// Test ParseSBOM with invalid JSON
	tmpFile, err := os.CreateTemp("", "invalid-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("{invalid}")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	err = usc.ParseSBOM(tmpFile.Name())
	assert.Error(t, err)

	// Test GetLatestGitTag error
	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("false") // command that returns error
	}
	defer func() { execCommand = exec.Command }()

	_, err = usc.getLatestGitTag("https://github.com/test/repo.git", "")
	assert.Error(t, err)
}
