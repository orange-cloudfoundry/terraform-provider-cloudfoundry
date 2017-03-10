package main

import (
	"errors"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/resources"
	"strings"
)

var descriptions map[string]string

func Provider() terraform.ResourceProvider {

	// The actual provider
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CF_API", ""),
				Description: "Your Cloud Foundry api url.",
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CF_USERNAME", ""),
				Description: "The username of an admin user. (Optional if you use an access token)",
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CF_PASSWORD", ""),
				Description: "The password of an admin user. (Optional if you use an access token)",
			},
			"enc_private_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CF_ENC_PRIVATE_KEY", ""),
				Description: "A GPG private key(s) generate from `gpg --export-secret-key -a <real name>` . Need a passphrase with 'enc_passphrase'.",
			},
			"enc_passphrase": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CF_ENC_PASSPHRASE", ""),
				Description: "The passphrase for your gpg key.",
			},
			"user_refresh_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The OAuth refresh token used to refresh your token.",
			},
			"user_access_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CF_TOKEN", ""),
				Description: "The OAuth token used to connect to a Cloud Foundry. (Optional if you use 'username' and 'password')",
			},
			"verbose": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Set to true to see request sent to Cloud Foundry.",
			},
			"skip_ssl_validation": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Set to true to skip verification of the API endpoint. Not recommended!",
			},
		},

		ResourcesMap: resources.RetrieveResourceMap(),

		ConfigureFunc: providerConfigure,
	}
}
func main() {
	cfProvider := &plugin.ServeOpts{ProviderFunc: Provider}
	plugin.Serve(cfProvider)
}
func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := cf_client.Config{
		ApiEndpoint:      d.Get("api_endpoint").(string),
		Username:         d.Get("username").(string),
		Password:         d.Get("password").(string),
		UserRefreshToken: parseToken(d.Get("user_refresh_token").(string)),
		UserAccessToken:  parseToken(d.Get("user_access_token").(string)),
		Locale:           "en_US",
		Verbose:          d.Get("verbose").(bool),
		SkipInsecureSSL:  d.Get("skip_ssl_validation").(bool),
		EncPrivateKey:    d.Get("enc_private_key").(string),
		Passphrase:       d.Get("enc_passphrase").(string),
	}
	if config.UserAccessToken == "" && (config.Username == "" || config.Password == "") {
		return nil, errors.New("You must provide an 'user_access_token' or an admin 'username' and 'password'")
	}
	if config.EncPrivateKey != "" && config.Passphrase == "" {
		return nil, errors.New("You must provide an 'enc_passphrase' to use a gpg key.")
	}
	return cf_client.NewCfClient(config)
}
func parseToken(token string) string {
	if token == "" {
		return ""
	}
	if strings.HasPrefix(token, "bearer ") {
		return token
	}
	return "bearer " + token
}
func init() {
	descriptions = map[string]string{
		"locale": "Set the locale for translation.",
	}
}
