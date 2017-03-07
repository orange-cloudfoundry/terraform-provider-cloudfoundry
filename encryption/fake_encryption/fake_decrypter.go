package fake_encryption

import "github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/encryption"

type FakeDecrypter struct {
}

func NewFakeDecrypter() encryption.Decrypter {
	return &FakeDecrypter{}
}
func (d FakeDecrypter) Decrypt(encString string) (string, error) {
	return encString, nil
}
