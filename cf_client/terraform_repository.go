package cf_client

import (
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"strings"
	"github.com/blang/semver"
	"code.cloudfoundry.org/cli/cf/models"
)

type TerraformRepository struct {
	configVersion            int
	target                   string
	apiVersion               string
	authorizationEndpoint    string
	loggregatorEndPoint      string
	dopplerEndPoint          string
	uaaEndpoint              string
	routingAPIEndpoint       string
	accessToken              string
	sshOAuthClient           string
	refreshToken             string
	sslDisabled              bool
	asyncTimeout             uint
	trace                    string
	colorEnabled             string
	locale                   string
	minCLIVersion            string
	minRecommendedCLIVersion string
}

// GETTERS

func (c *TerraformRepository) APIVersion() (string) {
	return c.apiVersion
}

func (c *TerraformRepository) AuthenticationEndpoint() (string) {
	return c.authorizationEndpoint
}

func (c *TerraformRepository) LoggregatorEndpoint() (string) {
	return c.loggregatorEndPoint
}

func (c *TerraformRepository) DopplerEndpoint() (string) {
	//revert this in v7.0, once CC advertise doppler endpoint, and
	//everyone has migrated from loggregator to doppler

	if c.dopplerEndPoint == "" {
		return strings.Replace(c.LoggregatorEndpoint(), "loggregator", "doppler", 1)
	}
	return c.dopplerEndPoint
}

func (c *TerraformRepository) UaaEndpoint() (string) {
	return c.uaaEndpoint
}

func (c *TerraformRepository) RoutingAPIEndpoint() (string) {
	return c.routingAPIEndpoint
}

func (c *TerraformRepository) APIEndpoint() (string) {
	return c.target
}

func (c *TerraformRepository) HasAPIEndpoint() (bool) {
	return c.apiVersion != "" && c.target != ""
}

func (c *TerraformRepository) AccessToken() (string) {
	return c.accessToken
}

func (c *TerraformRepository) SSHOAuthClient() (string) {
	return c.sshOAuthClient
}

func (c *TerraformRepository) RefreshToken() (string) {
	return c.refreshToken
}

func (c *TerraformRepository) OrganizationFields() (models.OrganizationFields) {
	return models.OrganizationFields{}
}

func (c *TerraformRepository) SpaceFields() (models.SpaceFields) {
	return models.SpaceFields{}
}

func (c *TerraformRepository) UserEmail() (string) {
	return coreconfig.NewTokenInfo(c.accessToken).Email
}

func (c *TerraformRepository) UserGUID() (string) {
	return coreconfig.NewTokenInfo(c.accessToken).UserGUID
}

func (c *TerraformRepository) Username() (string) {
	return coreconfig.NewTokenInfo(c.accessToken).Username
}

func (c *TerraformRepository) IsLoggedIn() (bool) {
	return c.accessToken != ""
}

func (c *TerraformRepository) HasOrganization() (bool) {
	return false
}

func (c *TerraformRepository) HasSpace() (bool) {
	return false
}

func (c *TerraformRepository) IsSSLDisabled() (bool) {
	return c.sslDisabled
}

func (c *TerraformRepository) IsMinAPIVersion(requiredVersion semver.Version) bool {
	var apiVersion string
	apiVersion = c.apiVersion

	actualVersion, err := semver.Make(apiVersion)
	if err != nil {
		return false
	}
	return actualVersion.GTE(requiredVersion)
}

func (c *TerraformRepository) IsMinCLIVersion(version string) bool {
	if version == "BUILT_FROM_SOURCE" {
		return true
	}
	var minCLIVersion string
	minCLIVersion = c.minCLIVersion
	if minCLIVersion == "" {
		return true
	}

	actualVersion, err := semver.Make(version)
	if err != nil {
		return false
	}
	requiredVersion, err := semver.Make(minCLIVersion)
	if err != nil {
		return false
	}
	return actualVersion.GTE(requiredVersion)
}

func (c *TerraformRepository) MinCLIVersion() (string) {
	return c.minCLIVersion
}

func (c *TerraformRepository) MinRecommendedCLIVersion() (string) {
	return c.minRecommendedCLIVersion
}

func (c *TerraformRepository) AsyncTimeout() (uint) {
	return c.asyncTimeout
}

func (c *TerraformRepository) Trace() (string) {
	return c.trace
}

func (c *TerraformRepository) ColorEnabled() (string) {
	return c.colorEnabled
}

func (c *TerraformRepository) Locale() (string) {
	return c.locale
}

func (c *TerraformRepository) PluginRepos() ([]models.PluginRepo) {
	return make([]models.PluginRepo, 0)
}

// SETTERS

func (c *TerraformRepository) ClearSession() {
	c.accessToken = ""
	c.refreshToken = ""

}

func (c *TerraformRepository) SetAPIEndpoint(endpoint string) {
	c.target = endpoint
}

func (c *TerraformRepository) SetAPIVersion(version string) {
	c.apiVersion = version
}

func (c *TerraformRepository) SetMinCLIVersion(version string) {
	c.minCLIVersion = version
}

func (c *TerraformRepository) SetMinRecommendedCLIVersion(version string) {
	c.minRecommendedCLIVersion = version
}

func (c *TerraformRepository) SetAuthenticationEndpoint(endpoint string) {
	c.authorizationEndpoint = endpoint
}

func (c *TerraformRepository) SetLoggregatorEndpoint(endpoint string) {
	c.loggregatorEndPoint = endpoint
}

func (c *TerraformRepository) SetDopplerEndpoint(endpoint string) {
	c.dopplerEndPoint = endpoint
}

func (c *TerraformRepository) SetUaaEndpoint(uaaEndpoint string) {
	c.uaaEndpoint = uaaEndpoint
}

func (c *TerraformRepository) SetRoutingAPIEndpoint(routingAPIEndpoint string) {
	c.routingAPIEndpoint = routingAPIEndpoint
}

func (c *TerraformRepository) SetAccessToken(token string) {
	c.accessToken = token
}

func (c *TerraformRepository) SetSSHOAuthClient(clientID string) {
	c.sshOAuthClient = clientID
}

func (c *TerraformRepository) SetRefreshToken(token string) {
	c.refreshToken = token
}

func (c *TerraformRepository) SetOrganizationFields(org models.OrganizationFields) {

}

func (c *TerraformRepository) SetSpaceFields(space models.SpaceFields) {

}

func (c *TerraformRepository) SetSSLDisabled(disabled bool) {
	c.sslDisabled = disabled
}

func (c *TerraformRepository) SetAsyncTimeout(timeout uint) {
	c.asyncTimeout = timeout
}

func (c *TerraformRepository) SetTrace(value string) {
	c.trace = value
}

func (c *TerraformRepository) SetColorEnabled(enabled string) {
	c.colorEnabled = enabled
}

func (c *TerraformRepository) SetLocale(locale string) {
	c.locale = locale
}

func (c *TerraformRepository) SetPluginRepo(repo models.PluginRepo) {
}

func (c *TerraformRepository) UnSetPluginRepo(index int) {

}
func (c *TerraformRepository) Close() {
}
func NewTerraformRepository() coreconfig.Repository {
	return &TerraformRepository{}
}
