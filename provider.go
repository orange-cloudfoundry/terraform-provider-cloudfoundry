package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"errors"
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
				DefaultFunc: schema.EnvDefaultFunc("CF_API", nil),
				Description: descriptions["api_endpoint"],
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				DefaultFunc: schema.EnvDefaultFunc("CF_USERNAME", nil),
				Description: descriptions["username"],
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				DefaultFunc: schema.EnvDefaultFunc("CF_PASSWORD", ""),
				Description: descriptions["password"],
			},
			"enc_private_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				DefaultFunc: schema.EnvDefaultFunc("CF_ENC_PRIVATE_KEY", nil),
				Description: descriptions["enc_private_key"],
			},
			"enc_passphrase": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				DefaultFunc: schema.EnvDefaultFunc("CF_ENC_PASSPHRASE", nil),
				Description: descriptions["enc_passphrase"],
			},
			"user_refresh_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["user_refresh_token"],
			},
			"user_access_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CF_TOKEN", ""),
				Description: descriptions["user_access_token"],
			},
			"locale": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "en_US",
				Description: descriptions["locale"],
			},
			"verbose": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions["verbose"],
			},
			"skip_ssl_validation": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions["skip_ssl_validation"],
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
		ApiEndpoint:        d.Get("api_endpoint").(string),
		Username:           d.Get("username").(string),
		Password:           d.Get("password").(string),
		UserRefreshToken:   parseToken(d.Get("user_refresh_token").(string)),
		UserAccessToken:    parseToken(d.Get("user_access_token").(string)),
		Locale:             d.Get("locale").(string),
		Verbose:            d.Get("verbose").(bool),
		SkipInsecureSSL:    d.Get("skip_ssl_validation").(bool),
		EncPrivateKey:      d.Get("enc_private_key").(string),
		Passphrase:         d.Get("enc_passphrase").(string),
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
		"password": "The password of an admin user. (Optional if you use an access token)",

		"username": "The username of an admin user. (Optional if you use an access token)",

		"api_endpoint": "Your Cloud Foundry api url.",

		"user_access_token": "The OAuth token used to connect to a Cloud Foundry. (Optional if you use 'username' and 'password')",

		"user_refresh_token": "The OAuth refresh token used to refresh your token.",

		"locale": "Set the locale for translation.",

		"verbose": "Set to true to see request sent to Cloud Foundry.",

		"skip_ssl_validation": "Set to true to skip verification of the API endpoint. Not recommended!",

		"enc_private_key": "One or multiple GPG private key(s) in base64. Need a passphrase with 'enc_passphrase'.",

		"enc_passphrase": "The passphrase for your gpg key.",
	}
}