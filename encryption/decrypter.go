package encryption

import (
	"encoding/base64"
	"bytes"
	"io/ioutil"
	"golang.org/x/crypto/openpgp"
	"strings"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
	"io"
)

type Decrypter struct {
	PrivateKey string
	Passphrase string
}

func (d Decrypter) sanitizeString(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\n")
	value = strings.Trim(value, "\r")
	return value
}

func (d Decrypter) Decrypt(encString string) (string, error) {

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
	entity.PrivateKey.Decrypt(passphraseByte)
	for _, subkey := range entity.Subkeys {
		subkey.PrivateKey.Decrypt(passphraseByte)
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
	bytes, err := ioutil.ReadAll(md.UnverifiedBody)
	if err != nil {
		return "", err
	}
	decStr := d.sanitizeString(string(bytes))

	return decStr, nil
}
