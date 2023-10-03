package giturl

import (
	"fmt"
)

type GitConfig struct {
	Private     bool
	Type        string
	Protocol    string
	Url         string
	Group       string
	ProjectName string
	AccessToken string
}

func GenCloneGitRepoUrl(g *GitConfig) string {
	return fmt.Sprintf("%s://oauth2:%s@%s/%s/%s", g.Protocol, g.AccessToken, g.Url, g.Group, g.ProjectName)
}
