package bitsmanager

import (
	"crypto/tls"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/common"
	"net/http"
)

type HttpHandler struct {
	SkipInsecureSSL bool
}

func (h HttpHandler) GetZipFile(path string) (FileHandler, error) {
	client := h.makeHttpClient()
	cleanFunc := func() error {
		return nil
	}
	resp, err := client.Get(path)
	if err != nil {
		return FileHandler{}, err
	}
	return FileHandler{
		ZipFile: resp.Body,
		Size:    resp.ContentLength,
		Clean:   cleanFunc,
	}, nil
}
func (h HttpHandler) Detect(path string) bool {
	return common.IsWebURL(path) && IsZipFile(path)
}
func (h HttpHandler) makeHttpClient() *http.Client {
	tr := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: h.SkipInsecureSSL},
	}
	return &http.Client{
		Transport: tr,
		Timeout:   0,
	}
}
func (h HttpHandler) GetSha1File(path string) (string, error) {
	client := h.makeHttpClient()
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return GetSha1FromReader(resp.Body)
}
