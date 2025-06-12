package entities

type DependencyInfo struct {
	Name    string
	Version string
	Type    string // "go-module", "github-action", etc.
	Source  string // path to source file where dependency is declared
}
