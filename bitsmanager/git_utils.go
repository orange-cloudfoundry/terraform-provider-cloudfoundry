package bitsmanager

import (
	"fmt"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"os"
	"strings"
)

type GitUtils struct {
	Folder     string
	Url        string
	RefName    string
	AuthMethod transport.AuthMethod
}

var refTypes []string = []string{"heads", "tags"}

func (g GitUtils) Clone() error {
	_, err := g.findRepo(false)
	if err != nil {
		return err
	}
	return nil
}
func (g GitUtils) GetCommitSha1() (string, error) {
	if g.refNameIsHash() {
		return g.RefName, nil
	}
	repo, err := g.findRepo(true)
	if err != nil {
		return "", err
	}
	iter, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return "", err
	}
	defer iter.Close()
	commit, err := iter.Next()
	if err != nil {
		return "", err
	}
	return commit.Hash.String(), nil
}
func (g GitUtils) refNameIsHash() bool {
	return len(g.RefName) == 40
}
func (g GitUtils) findRepoFromHash(isBare bool) (*git.Repository, error) {
	repo, err := git.PlainClone(g.Folder, isBare, &git.CloneOptions{
		URL:  g.Url,
		Auth: g.AuthMethod,
	})
	if err != nil {
		return nil, err
	}
	tree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	err = tree.Checkout(&git.CheckoutOptions{
		Hash:  plumbing.NewHash(g.RefName),
		Force: true,
	})
	if err != nil {
		return nil, err
	}
	return repo, nil
}
func (g GitUtils) findRepo(isBare bool) (repo *git.Repository, err error) {
	if g.refNameIsHash() {
		repo, err = g.findRepoFromHash(isBare)
		return
	}
	for _, refType := range refTypes {
		repo, err = git.PlainClone(g.Folder, isBare, &git.CloneOptions{
			URL:          g.Url,
			SingleBranch: true,
			Auth:         g.AuthMethod,
			ReferenceName: plumbing.ReferenceName(fmt.Sprintf(
				"refs/%s/%s",
				refType,
				strings.ToLower(g.RefName),
			)),
			Depth: 1,
		})
		if err == nil {
			return
		}
		if err.Error() == "reference not found" {
			os.RemoveAll(g.Folder)
			os.Mkdir(g.Folder, 0777)
			continue
		}
		return
	}
	return
}
