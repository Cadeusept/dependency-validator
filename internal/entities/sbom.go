package entities

// SBOM represents a CycloneDX Software Bill of Materials
type SBOM struct {
	BomFormat    string       `json:"bomFormat"`
	SpecVersion  string       `json:"specVersion"`
	Components   []Component  `json:"components"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
	Metadata     Metadata     `json:"metadata"`
}

type Component struct {
	Type       string              `json:"type"`
	Name       string              `json:"name"`
	Version    string              `json:"version,omitempty"`
	Purl       string              `json:"purl,omitempty"`
	Properties []ComponentProperty `json:"properties,omitempty"`
}

type ComponentProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Dependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn"`
}

type Metadata struct {
	Timestamp string `json:"timestamp"`
	Tools     struct {
		Components []ToolComponent `json:"components"`
	} `json:"tools"`
}

type ToolComponent struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version"`
}
