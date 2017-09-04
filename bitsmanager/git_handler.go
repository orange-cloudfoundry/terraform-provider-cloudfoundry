package bitsmanager

import (
	"crypto/tls"
	"fmt"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/common"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/client"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type GitHandler struct {
}

func NewGitHandler(skipInsecureSSL bool) *GitHandler {
	customClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: skipInsecureSSL},
		},
	}
	client.InstallProtocol(
		"https",
		githttp.NewClient(customClient),
	)
	return &GitHandler{}
}
func (h GitHandler) GetZipFile(path string) (FileHandler, error) {
	tmpDir, err := ioutil.TempDir("", "git-tf")
	if err != nil {
		return FileHandler{}, err
	}
	gitUtils := h.makeGitUtils(tmpDir, path)
	err = gitUtils.Clone()
	if err != nil {
		return FileHandler{}, err
	}
	err = os.RemoveAll(filepath.Join(tmpDir, ".git"))
	localFh, err := NewLocalHandler().GetZipFile(tmpDir)
	if err != nil {
		return FileHandler{}, err
	}
	cleanFunc := func() error {
		err := localFh.Clean()
		if err != nil {
			return err
		}
		return os.RemoveAll(tmpDir)
	}
	return FileHandler{
		ZipFile: localFh.ZipFile,
		Size:    localFh.Size,
		Clean:   cleanFunc,
	}, nil
}
func (h GitHandler) makeGitUtils(tmpDir, path string) *GitUtils {
	u, _ := url.Parse(path)

	refName := "master"
	if u.Fragment != "" {
		refName = u.Fragment
	}
	var authMethod transport.AuthMethod
	if u.User != nil {
		password, _ := u.User.Password()
		authMethod = githttp.NewBasicAuth(u.User.Username(), password)
	}
	gitUtils := &GitUtils{
		Url:        fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path),
		Folder:     tmpDir,
		RefName:    refName,
		AuthMethod: authMethod,
	}
	return gitUtils
}
func (h GitHandler) GetSha1File(path string) (string, error) {
	tmpDir, err := ioutil.TempDir("", "git-tf")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	gitUtils := h.makeGitUtils(tmpDir, path)
	return gitUtils.GetCommitSha1()
}
func (h GitHandler) Detect(path string) bool {
	if !common.IsWebURL(path) {
		return false
	}
	u, err := url.Parse(path)
	if err != nil {
		return false
	}
	return HasExtFile(u.Path, ".git")
}
