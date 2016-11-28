package encryption

import (
	"encoding/base64"
	"bytes"
	"io/ioutil"
	"golang.org/x/crypto/openpgp"
	"strings"
)

type Decrypter struct {
	PrivateKey string
	Passphrase string
}

func (d Decrypter) getPrivateKeyBase64Decode() ([]byte, error) {
	if d.PrivateKey == "" {
		return make([]byte, 0), nil
	}
	return base64.StdEncoding.DecodeString(d.sanitizeString(d.PrivateKey))
}
func (d Decrypter) sanitizeString(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\n")
	value = strings.Trim(value, "\r")
	return value
}

func (d Decrypter) Decrypt(encString string) (string, error) {
	privateKey, err := d.getPrivateKeyBase64Decode()
	if err != nil {
		return "", err
	}
	if len(privateKey) == 0 {
		return encString, nil
	}
	// init some vars
	var entity *openpgp.Entity
	var entityList openpgp.EntityList

	// Open the private key file
	buf := bytes.NewBuffer(privateKey)
	entityList, err = openpgp.ReadKeyRing(buf)
	if err != nil {
		return "", err
	}
	entity = entityList[0]

	// Get the passphrase and read the private key.
	// Have not touched the encrypted string yet
	passphraseByte := []byte(d.Passphrase)
	entity.PrivateKey.Decrypt(passphraseByte)
	for _, subkey := range entity.Subkeys {
		subkey.PrivateKey.Decrypt(passphraseByte)
	}
	// Decode the base64 string
	dec, err := base64.StdEncoding.DecodeString(encString)
	if err != nil {
		return encString, nil
	}
	// Decrypt it with the contents of the private key
	md, err := openpgp.ReadMessage(bytes.NewBuffer(dec), entityList, nil, nil)
	if err != nil {
		return encString, nil
	}
	bytes, err := ioutil.ReadAll(md.UnverifiedBody)
	if err != nil {
		return "", err
	}
	decStr := d.sanitizeString(string(bytes))

	return decStr, nil
}
