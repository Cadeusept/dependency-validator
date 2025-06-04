package entities

type Repo struct {
	Name    string `yaml:"name"`
	RepoURL string `yaml:"repo_url"`
	Token   string `yaml:"token,omitempty"`
}
