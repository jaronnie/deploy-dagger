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
	// TODO: support github or other by type field
	if g.Private && g.Type == "github" {
		return fmt.Sprintf("%s://%s@%s/%s/%s", g.Protocol, g.AccessToken, g.Url, g.Group, g.ProjectName)
	} else if g.Private && g.Type == "gitlab" {
		return fmt.Sprintf("%s://oauth2:%s@%s/%s/%s", g.Protocol, g.AccessToken, g.Url, g.Group, g.ProjectName)
	}
	return fmt.Sprintf("%s://%s/%s/%s", g.Protocol, g.Url, g.Group, g.ProjectName)
}
