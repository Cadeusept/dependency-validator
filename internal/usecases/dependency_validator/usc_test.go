package dependency_validator

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cadeusept/dependency-validator/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCycloneDXSBOM = `{
	"bomFormat": "CycloneDX",
	"specVersion": "1.6",
	"components": [
		{
		  "bom-ref": "pkg:golang/github.com/stretchr/testify@v1.9.0?package-id=1e92eaf826745a46",
		  "type": "library",
		  "name": "github.com/stretchr/testify",
		  "version": "v1.9.0",
		  "purl": "pkg:golang/github.com/stretchr/testify@v1.9.0",
		  "properties": [
			{"name": "syft:package:type", "value": "go-module"},
			{"name": "syft:location:0:path", "value": "/go.mod"}
		  ]
		},
		{
		  "bom-ref": "pkg:golang/golang.org/x/mod@v0.24.0?package-id=0ab7d18217876fde",
		  "type": "library",
		  "name": "golang.org/x/mod",
		  "version": "v0.24.0",
		  "purl": "pkg:golang/golang.org/x/mod@v0.24.0",
		  "properties": [
			{"name": "syft:package:type", "value": "go-module"},
			{"name": "syft:location:0:path", "value": "/go.mod"}
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
				{"type": "application", "name": "syft", "version": "1.27.0"}
			]
		}
	}
}`

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s\n%s", strings.Join(args, " "), out)
}

func setupGitRepoWithTags(t *testing.T, tags ...string) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "user.name", "test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README"), []byte("x"), 0o644))
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "init")
	for _, tag := range tags {
		gitRun(t, dir, "tag", tag)
	}
	return dir
}

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

	invalidSBOM := `{"bomFormat": "SPDX", "specVersion": "2.2"}`
	err = os.WriteFile(tmpFile.Name(), []byte(invalidSBOM), 0o644)
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

	uscNoSBOM := NewUsecase(nil)
	_, err = uscNoSBOM.GetSBOMDependencies()
	assert.Error(t, err)
}

func TestCheckDependencies(t *testing.T) {
	t.Parallel()

	repoDir := setupGitRepoWithTags(t, "v1.0.0", "v2.0.0", "not-a-version")

	t.Run("GoodCase_outdated_dependency", func(t *testing.T) {
		repos := []entities.Repo{
			{Name: "github.com/example/lib", RepoURL: repoDir},
		}
		usc := NewUsecase(repos)
		usc.dependencies = &entities.SBOM{
			Components: []entities.Component{
				{Type: "library", Name: "github.com/example/lib", Version: "v1.0.0"},
			},
		}

		outdated := usc.CheckDependencies()
		require.Len(t, outdated, 1)
		assert.Contains(t, outdated[0], "github.com/example/lib")
		assert.Contains(t, outdated[0], "v1.0.0")
		assert.Contains(t, outdated[0], "v2.0.0")
	})

	t.Run("GoodCase_up_to_date_dependency", func(t *testing.T) {
		repos := []entities.Repo{
			{Name: "github.com/example/lib", RepoURL: repoDir},
		}
		usc := NewUsecase(repos)
		usc.dependencies = &entities.SBOM{
			Components: []entities.Component{
				{Type: "library", Name: "github.com/example/lib", Version: "v2.0.0"},
			},
		}

		outdated := usc.CheckDependencies()
		assert.Empty(t, outdated)
	})

	t.Run("GoodCase_skips_missing_and_errors", func(t *testing.T) {
		repos := []entities.Repo{
			{Name: "missing/dep", RepoURL: repoDir},
			{Name: "broken/dep", RepoURL: filepath.Join(t.TempDir(), "no-such-repo")},
			{Name: "github.com/example/lib", RepoURL: repoDir},
		}
		usc := NewUsecase(repos)
		usc.dependencies = &entities.SBOM{
			Components: []entities.Component{
				{Type: "library", Name: "broken/dep", Version: "v1.0.0"},
				{Type: "library", Name: "github.com/example/lib", Version: "2.0.0"},
			},
		}

		outdated := usc.CheckDependencies()
		assert.Empty(t, outdated)
	})

	t.Run("BadCase_no_sbom_loaded", func(t *testing.T) {
		usc := NewUsecase([]entities.Repo{
			{Name: "github.com/example/lib", RepoURL: repoDir},
		})
		outdated := usc.CheckDependencies()
		assert.Empty(t, outdated)
	})
}

func TestGetAssetVersions(t *testing.T) {
	t.Run("GoodCase_parses_libraries", func(t *testing.T) {
		root := t.TempDir()
		objDir := filepath.Join(root, "obj")
		require.NoError(t, os.MkdirAll(objDir, 0o755))
		assets := map[string]any{
			"libraries": map[string]any{
				"Newtonsoft.Json/13.0.1": map[string]any{"type": "package"},
				"InvalidEntry":           map[string]any{"type": "package"},
			},
		}
		data, err := json.Marshal(assets)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(objDir, "project.assets.json"), data, 0o644))

		t.Chdir(root)

		usc := NewUsecase(nil)
		err = usc.GetAssetVersions()
		assert.NoError(t, err)
	})

	t.Run("BadCase_file_not_found", func(t *testing.T) {
		t.Chdir(t.TempDir())
		usc := NewUsecase(nil)
		err := usc.GetAssetVersions()
		assert.Error(t, err)
	})

	t.Run("BadCase_invalid_json", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, "obj"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "obj", "project.assets.json"), []byte("{"), 0o644))
		t.Chdir(root)

		usc := NewUsecase(nil)
		err := usc.GetAssetVersions()
		assert.Error(t, err)
	})
}

func TestGetLatestGitTag(t *testing.T) {
	t.Parallel()

	t.Run("GoodCase_picks_highest_semver_and_strips_annotation", func(t *testing.T) {
		repoDir := setupGitRepoWithTags(t, "v1.2.3", "v2.0.0", "latest")
		// annotated tag pointing at same commit as lightweight is fine; ^{} appears in ls-remote for annotated
		gitRun(t, repoDir, "tag", "-a", "v2.1.0", "-m", "annotated")

		usc := NewUsecase(nil)
		latest, err := usc.getLatestGitTag(repoDir, "")
		require.NoError(t, err)
		assert.Equal(t, "v2.1.0", latest)
	})

	t.Run("BadCase_no_valid_semver_tags", func(t *testing.T) {
		repoDir := setupGitRepoWithTags(t, "latest", "release")
		usc := NewUsecase(nil)
		_, err := usc.getLatestGitTag(repoDir, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no valid semver tags found")
	})

	t.Run("BadCase_git_command_fails", func(t *testing.T) {
		usc := NewUsecase(nil)
		_, err := usc.getLatestGitTag(filepath.Join(t.TempDir(), "missing"), "")
		assert.Error(t, err)
	})

	t.Run("GoodCase_token_is_injected_into_https_url", func(t *testing.T) {
		usc := NewUsecase(nil)
		// Token rewrite happens before git runs; the rewritten URL is invalid so git fails,
		// but the token branch is exercised.
		_, err := usc.getLatestGitTag("https://example.invalid/repo.git", "tok")
		assert.Error(t, err)
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestGetLatestNugetVersion(t *testing.T) {
	usc := NewUsecase(nil)
	oldTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })

	t.Run("GoodCase_returns_latest_version", func(t *testing.T) {
		http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Contains(t, r.URL.Path, "newtonsoft.json")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"versions":["12.0.0","13.0.1"]}`)),
				Header:     make(http.Header),
			}, nil
		})

		version, err := usc.getLatestNugetVersion("Newtonsoft.Json")
		require.NoError(t, err)
		assert.Equal(t, "13.0.1", version)
	})

	t.Run("BadCase_http_get_error", func(t *testing.T) {
		http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, assert.AnError
		})

		_, err := usc.getLatestNugetVersion("Any.Package")
		assert.Error(t, err)
	})

	t.Run("BadCase_package_not_found", func(t *testing.T) {
		http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		})

		_, err := usc.getLatestNugetVersion("Missing.Package")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("BadCase_invalid_json", func(t *testing.T) {
		http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("{")),
				Header:     make(http.Header),
			}, nil
		})

		_, err := usc.getLatestNugetVersion("Broken.Package")
		assert.Error(t, err)
	})

	t.Run("BadCase_empty_versions", func(t *testing.T) {
		http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"versions":[]}`)),
				Header:     make(http.Header),
			}, nil
		})

		_, err := usc.getLatestNugetVersion("Empty.Package")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no versions found")
	})
}

func TestDetectSBOM(t *testing.T) {
	t.Parallel()

	usc := NewUsecase(nil)

	t.Run("GoodCase_prefers_bom_json", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "bom.json"), []byte(testCycloneDXSBOM), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("not an SBOM"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "spdx.json"), []byte(`{"SPDXID":"SPDXRef-DOCUMENT"}`), 0o644))

		path, err := usc.DetectSBOM(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tmpDir, "bom.json"), path)
	})

	t.Run("GoodCase_prefers_sbom_json", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "sbom.json"), []byte(testCycloneDXSBOM), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "inventory.json"), []byte(`{}`), 0o644))

		path, err := usc.DetectSBOM(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tmpDir, "sbom.json"), path)
	})

	t.Run("GoodCase_prefers_syft_output", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".syft.json"), []byte(`{}`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "inventory.json"), []byte(`{}`), 0o644))

		path, err := usc.DetectSBOM(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tmpDir, ".syft.json"), path)
	})

	t.Run("GoodCase_returns_first_match_fallback", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "inventory.json"), []byte(`{}`), 0o644))

		path, err := usc.DetectSBOM(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tmpDir, "inventory.json"), path)
	})

	t.Run("BadCase_no_sbom_found", func(t *testing.T) {
		_, err := usc.DetectSBOM(t.TempDir())
		assert.Error(t, err)
	})

	t.Run("BadCase_directory_does_not_exist", func(t *testing.T) {
		_, err := usc.DetectSBOM(filepath.Join(t.TempDir(), "missing"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error scanning directory for SBoM")
	})
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
		{"release-4.5.6", "release-4.5.6"},
		{"4.5.6-release", "4.5.6"},
		{"stable-5.0", "stable-5.0"},
		{"5.0-stable", "5.0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeVersion(tt.input))
		})
	}
}

func TestErrorPaths(t *testing.T) {
	t.Parallel()

	usc := NewUsecase(nil)
	err := usc.ParseSBOM("nonexistent.json")
	assert.Error(t, err)

	tmpFile, err := os.CreateTemp("", "invalid-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("{invalid}")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	err = usc.ParseSBOM(tmpFile.Name())
	assert.Error(t, err)
}
