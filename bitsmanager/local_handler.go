package bitsmanager

import (
	"code.cloudfoundry.org/cli/cf/appfiles"
	"io/ioutil"
	"os"
)

type LocalHandler struct {
}

func (h LocalHandler) GetZipFile(path string) (FileHandler, error) {
	zipFile, err := ioutil.TempFile("", "uploads")
	if err != nil {
		return FileHandler{}, err
	}
	zipper := appfiles.ApplicationZipper{}
	err = zipper.Zip(path, zipFile)
	if err != nil {
		return FileHandler{}, err
	}
	file, err := os.Open(zipFile.Name())
	cleanFunc := func() error {
		return os.Remove(zipFile.Name())
	}
	fs, _ := file.Stat()
	return FileHandler{
		ZipFile: file,
		Size:    fs.Size(),
		Clean:   cleanFunc,
	}, err
}
func (h LocalHandler) Detect(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
func (h LocalHandler) GetSha1File(path string) (string, error) {
	fileHandler, err := h.GetZipFile(path)
	if err != nil {
		return "", err
	}
	defer fileHandler.ZipFile.Close()
	return GetSha1FromReader(fileHandler.ZipFile)
}
