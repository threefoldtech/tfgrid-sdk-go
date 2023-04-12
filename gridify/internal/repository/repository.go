// package repository for getting repository information
package repository

import (
	"github.com/pkg/errors"

	"github.com/go-git/go-git/v5"
)

// GetRepositoryURL returns the repository remote URL
func GetRepositoryURL(path string) (string, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return "", errors.Wrap(err, "failed to open repository ")

	}

	repoConfig, err := repo.Config()
	if err != nil {
		return "", errors.Wrap(err, "failed to get repository config")

	}
	remote, ok := repoConfig.Remotes["origin"]
	if !ok {
		return "", errors.New("no repository remote origin found")
	}

	if len(remote.URLs) == 0 {
		return "", errors.New("no remote origin urls found")
	}
	repoURL := remote.URLs[0]
	return repoURL, nil
}
