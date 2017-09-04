package bitsmanager

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"github.com/viant/toolbox"
	"io"
	"path/filepath"
	"strings"
)

var ZIP_FILE_EXT []string = []string{
	".zip",
	".jar",
}
var TAR_FILE_EXT []string = []string{
	".tar",
}
var GZIP_FILE_EXT []string = []string{
	".gz",
	".gzip",
}
var TARGZ_FILE_EXT []string = []string{
	".tgz",
}

func GetSha1FromReader(reader io.ReadCloser) (string, error) {
	buf := new(bytes.Buffer)
	_, err := io.CopyN(buf, reader, CHUNK_FOR_SHA1)
	if err != nil && err != io.EOF {
		return "", err
	}
	// we don't want to retrieve everything
	reader.Close()

	h := sha1.New()
	h.Write(buf.Bytes())
	return base64.URLEncoding.EncodeToString(h.Sum(nil)), nil
}
func HasExtFile(path string, extensions ...string) bool {
	ext := filepath.Ext(path)
	if ext == "" {
		return false
	}
	return toolbox.HasSliceAnyElements(extensions, strings.ToLower(ext))
}
func IsZipFile(path string) bool {
	return HasExtFile(path, ZIP_FILE_EXT...)
}
func IsTarFile(path string) bool {
	return HasExtFile(path, TAR_FILE_EXT...)
}
func IsTarGzFile(path string) bool {
	isTgz := HasExtFile(path, TARGZ_FILE_EXT...)
	if isTgz {
		return true
	}
	isGz := HasExtFile(path, GZIP_FILE_EXT...)
	if !isGz {
		return false
	}
	return IsTarFile(filepath.Ext(strings.TrimSuffix(path, filepath.Ext(path))))
}
