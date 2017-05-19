package encryption

import (
	"bytes"
	"encoding/base64"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
	"io"
	"io/ioutil"
	"strings"
)

type Decrypter interface {
	Decrypt(encString string) (string, error)
}

type PgpDecrypter struct {
	PrivateKey string
	Passphrase string
}

func NewPgpDecrypter(privateKey, passphrase string) Decrypter {
	return &PgpDecrypter{
		PrivateKey: privateKey,
		Passphrase: passphrase,
	}
}
func (d PgpDecrypter) sanitizeString(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\n")
	value = strings.Trim(value, "\r")
	return value
}

func (d PgpDecrypter) Decrypt(encString string) (string, error) {

	// init some vars
	var entity *openpgp.Entity
	// Open the private key file
	buf := bytes.NewBuffer([]byte(d.PrivateKey))
	blockPrivateKey, err := armor.Decode(buf)
	if err != nil {
		return encString, nil
	}
	packetDecoder := packet.NewReader(blockPrivateKey.Body)
	entity, err = openpgp.ReadEntity(packetDecoder)

	// Get the passphrase and read the private key.
	// Have not touched the encrypted string yet
	passphraseByte := []byte(d.Passphrase)
	err = entity.PrivateKey.Decrypt(passphraseByte)
	if err != nil {
		return "", err
	}
	for _, subkey := range entity.Subkeys {
		err = subkey.PrivateKey.Decrypt(passphraseByte)
		if err != nil {
			return "", err
		}
	}
	// Decode the base64 string
	var dec io.Reader

	blockPublicKey, err := armor.Decode(bytes.NewBuffer([]byte(encString)))
	if err != nil {
		decBase64, err := base64.StdEncoding.DecodeString(encString)
		if err != nil {
			return encString, nil
		}
		dec = bytes.NewBuffer(decBase64)
	} else {
		dec = blockPublicKey.Body
	}

	// Decrypt it with the contents of the private key

	md, err := openpgp.ReadMessage(dec, openpgp.EntityList([]*openpgp.Entity{entity}), nil, nil)
	if err != nil {
		return encString, nil
	}
	b, err := ioutil.ReadAll(md.UnverifiedBody)
	if err != nil {
		return "", err
	}
	decStr := d.sanitizeString(string(b))

	return decStr, nil
}
