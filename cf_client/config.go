package cf_client

type Config struct {
	ApiEndpoint      string
	SkipInsecureSSL  bool
	Username         string
	Password         string
	UserRefreshToken string
	UserAccessToken  string
	Locale           string
	Verbose          bool
	EncPrivateKey    string
	Passphrase       string
}

func (c *Config) SkipSSLValidation() bool {
	return c.SkipInsecureSSL
}

func (c *Config) Target() string {
	return c.ApiEndpoint
}
func (c *Config) ClientID() string {
	return c.Username
}
func (c *Config) ClientSecret() string {
	return c.Password
}

func (c *Config) AccessToken() string {
	return c.UserAccessToken
}
func (c *Config) RefreshToken() string {
	return c.UserRefreshToken
}
func (c *Config) SetAccessToken(token string) {
	c.UserAccessToken = token
}
