package encryption_test

import (
	. "github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/encryption"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Encryption", func() {
	encPrivKey := `-----BEGIN PGP PRIVATE KEY BLOCK-----
Version: GnuPG v2

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
=ZraR
-----END PGP PRIVATE KEY BLOCK-----`

	encMessageB64 := "hQEMAxBftgU+yseEAQf/TIhwU6XhiSJX3XURcbwZGYlU1BAkXMKXyNqdyqcdPw3oLez2kYxNYA/Paqa5wKXW2pnlLLgxJ33WpjakTPLp/59jmfAz99tx4ry4aCixTQlti0ApeUtBFTlK4hnyk2L3Imu3LOVOuGpW8ItYIR5wjYdVS4fGbH2apB9GbLAAEDYo7ueym8DsLH8lyArorjENPbsbDKrClZnuuf3UV8Mwdg8X3/8CtxFEB7ehC0oUwB6JFM1HdkW5UdpVxsb6E3rkM84rnjQ5tUZGYhOENNOG27QkNOyxtHCzjd4SR2bxCYzxW+3OCHF7MwVX1dpTvnX00LWJXDJ95aRh/XzyeTpOitJGAfkNh84OEcuqAlYcoN9LQv3zrE3RjtGDfViq5LIswiRuSZ9cXQ4s/73xSglK6dzYD2Mdq4wvA5hnih0W+CZXIie38Q5nLQ=="
	encMessageArmored := `-----BEGIN PGP MESSAGE-----
Version: GnuPG v2

hQEMAxBftgU+yseEAQgAmQ1bt9CbCIffC30WOXEyvlRU4pZWpUiAi5XEQUay0nID
VTtBuDg46npv+EDUHIjhXm82ZcS0Yz/RG+rjaMgLqOfNre3joLqkW6mykGO35itM
scTrD7oolRiM7uBS/w9UycoulXjptuPznM+xcWMPV/vOC+At7TmYSYdIbxFwTsDI
zwPBA9y8GwMidBW27e5jikInYa8YJIQslYzzGNLFtIUUYcwTXC4j9dAvi/bXF6F3
C9/5aDv8chRi1OrPkPNhtKOIb23gbRXwkgxZTUG1UATDIsXcjcW3VHrLVKkqm3qt
ZIa5o0reuzX2E7BhCKWEigGh010VymSYtIJn4+q5XdJGAck5KnX21bq19Q1H51p4
cZuZCVNDO/OHnzbxwh1YGpc+P+6afVq7ZABd5BqeT9s9J8W2dYkBXXxfNdMnAudx
s6aRDLpAwg==
=G86K
-----END PGP MESSAGE-----`
	passphrase := "ahaletcf"

	Context("When want to decrypt a message", func() {
		It("should decrypt it if a private key was given and message in base64", func() {
			decrypter := Decrypter{
				PrivateKey: encPrivKey,
				Passphrase: passphrase,
			}
			decMessage, err := decrypter.Decrypt(encMessageB64)
			Expect(err).ToNot(HaveOccurred())
			Expect(decMessage).Should(Equal("mypassword"))
		})
		It("should decrypt it if a private key was given and message is armored", func() {
			decrypter := Decrypter{
				PrivateKey: encPrivKey,
				Passphrase: passphrase,
			}
			decMessage, err := decrypter.Decrypt(encMessageArmored)
			Expect(err).ToNot(HaveOccurred())
			Expect(decMessage).Should(Equal("mypassword"))
		})
		It("should send back message without decrypting it if a private key was not given", func() {
			decrypter := Decrypter{
				Passphrase: passphrase,
			}
			decMessage, err := decrypter.Decrypt(encMessageB64)
			Expect(err).ToNot(HaveOccurred())
			Expect(decMessage).Should(Equal(encMessageB64))
		})
		It("should send back message without decrypting it if message is not encrypted", func() {
			message := "mymessage"
			decrypter := Decrypter{
				PrivateKey: encPrivKey,
				Passphrase: passphrase,
			}
			decMessage, err := decrypter.Decrypt(message)
			Expect(err).ToNot(HaveOccurred())
			Expect(decMessage).Should(Equal(message))
		})
	})
})
