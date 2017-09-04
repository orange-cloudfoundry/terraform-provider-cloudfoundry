package bitsmanager

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/common"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type HttpHandler struct {
	SkipInsecureSSL bool
}

func NewHttpHandler(skipInsecureSSL bool) *HttpHandler {
	return &HttpHandler{skipInsecureSSL}
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
	err = h.checkRespHttpError(resp)
	if err != nil {
		return FileHandler{}, err
	}
	if IsTarFile(path) {
		defer resp.Body.Close()
		return h.tar2Zip(resp.Body)
	}
	if IsTarGzFile(path) {
		defer resp.Body.Close()
		return h.targz2Zip(resp.Body)
	}
	return FileHandler{
		ZipFile: resp.Body,
		Size:    resp.ContentLength,
		Clean:   cleanFunc,
	}, nil
}
func (h HttpHandler) checkRespHttpError(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	content := ""
	if err == nil {
		content = string(b)
	}
	return fmt.Errorf(
		"Error occured when dowloading file: %d %s: \n%s",
		resp.StatusCode,
		http.StatusText(resp.StatusCode),
		content,
	)
}
func (h HttpHandler) Detect(path string) bool {
	return common.IsWebURL(path) && (IsZipFile(path) || IsTarFile(path) || IsTarGzFile(path))
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
func (h HttpHandler) targz2Zip(r io.ReadCloser) (FileHandler, error) {
	gzf, err := gzip.NewReader(r)
	if err != nil {
		return FileHandler{}, err
	}
	return h.tar2Zip(gzf)
}
func (h HttpHandler) tar2Zip(r io.ReadCloser) (FileHandler, error) {
	zipFile, err := ioutil.TempFile("", "downloads-tf")
	if err != nil {
		return FileHandler{}, err
	}
	cleanFunc := func() error {
		return os.Remove(zipFile.Name())
	}
	err = h.writeTarToZip(r, zipFile)
	if err != nil {
		zipFile.Close()
		return FileHandler{}, err
	}
	zipFile.Close()
	file, err := os.Open(zipFile.Name())
	if err != nil {
		return FileHandler{}, err
	}
	fs, _ := file.Stat()
	return FileHandler{
		ZipFile: file,
		Size:    fs.Size(),
		Clean:   cleanFunc,
	}, nil
}
func (h HttpHandler) writeTarToZip(r io.Reader, zipFile *os.File) error {
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	tarReader := tar.NewReader(r)
	hasRootFolder := false
	i := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		fileInfo := header.FileInfo()
		if i == 0 && fileInfo.IsDir() {
			hasRootFolder = true
			continue
		}
		zipHeader, err := zip.FileInfoHeader(fileInfo)
		if err != nil {
			return err
		}
		if !hasRootFolder {
			zipHeader.Name = header.Name
		} else {
			splitFile := strings.Split(header.Name, "/")
			zipHeader.Name = strings.Join(splitFile[1:], "/")
		}
		if !fileInfo.IsDir() {
			zipHeader.Method = zip.Deflate
		}
		w, err := zipWriter.CreateHeader(zipHeader)
		if err != nil {
			return err
		}
		i++
		if fileInfo.IsDir() {
			continue
		}
		_, err = io.Copy(w, tarReader)
	}
	return nil
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
	err = h.checkRespHttpError(resp)
	if err != nil {
		return "", err
	}
	return GetSha1FromReader(resp.Body)
}
