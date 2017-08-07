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

func IsZipFile(path string) bool {
	ext := filepath.Ext(path)
	if ext == "" {
		return false
	}
	return toolbox.HasSliceAnyElements(ZIP_FILE_EXT, strings.ToLower(ext))
}
