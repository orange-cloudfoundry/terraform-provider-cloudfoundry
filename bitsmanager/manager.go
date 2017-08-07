package bitsmanager

import (
	"fmt"
	"io"
)

const (
	CHUNK_FOR_SHA1 = 5 * 1024
	APP_FILENAME   = "application.zip"
)

type Handler interface {
	GetZipFile(path string) (fileHandler FileHandler, err error)
	GetSha1File(path string) (sha1 string, err error)
	Detect(path string) bool
}
type FileHandler struct {
	ZipFile io.ReadCloser
	Size    int64
	Clean   func() error
}
type BitsManager interface {
	Upload(appGuid string, path string) error
	GetSha1(path string) (sha1 string, err error)
	IsDiff(path string, currentSha1 string) (isDiff bool, sha1 string, err error)
}

type CloudControllerBitsManager struct {
	appBitsRepo ApplicationBitsRepository
	handlers    []Handler
}

func NewCloudControllerBitsManager(appBitsRepo ApplicationBitsRepository, handlers []Handler) (manager CloudControllerBitsManager) {
	manager.appBitsRepo = appBitsRepo
	manager.handlers = handlers
	return
}
func (m CloudControllerBitsManager) GetSha1(path string) (string, error) {
	h, err := m.chooseHandler(path)
	if err != nil {
		return "", err
	}
	return h.GetSha1File(path)
}
func (m CloudControllerBitsManager) Upload(appGuid string, path string) error {
	h, err := m.chooseHandler(path)
	if err != nil {
		return err
	}
	fileHandler, err := h.GetZipFile(path)
	if err != nil {
		return err
	}
	defer fileHandler.ZipFile.Close()
	defer fileHandler.Clean()
	return m.appBitsRepo.UploadBits(appGuid, fileHandler.ZipFile, fileHandler.Size)
}
func (m CloudControllerBitsManager) IsDiff(path string, currentSha1 string) (bool, string, error) {
	h, err := m.chooseHandler(path)
	if err != nil {
		return true, "", err
	}
	sha1Given, err := h.GetSha1File(path)
	if err != nil {
		return true, "", err
	}
	return currentSha1 != sha1Given, sha1Given, nil
}
func (m CloudControllerBitsManager) chooseHandler(path string) (Handler, error) {
	for _, h := range m.handlers {
		if h.Detect(path) {
			return h, nil
		}
	}
	return nil, fmt.Errorf("Handler for path '%s' cannot be found.", path)
}
