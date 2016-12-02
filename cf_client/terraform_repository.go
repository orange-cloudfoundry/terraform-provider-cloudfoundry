package cf_client

import (
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"strings"
	"github.com/blang/semver"
	"code.cloudfoundry.org/cli/cf/models"
	"sync"
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
	mutex                    *sync.RWMutex
}

// GETTERS

func (c *TerraformRepository) APIVersion() (apiVersion string) {
	c.read(func() {
		apiVersion = c.apiVersion
	})
	return
}

func (c *TerraformRepository) AuthenticationEndpoint() (authEndpoint string) {
	c.read(func() {
		authEndpoint = c.authorizationEndpoint
	})
	return
}

func (c *TerraformRepository) LoggregatorEndpoint() (logEndpoint string) {
	c.read(func() {
		logEndpoint = c.loggregatorEndPoint
	})
	return
}

func (c *TerraformRepository) DopplerEndpoint() (dopplerEndpoint string) {
	//revert this in v7.0, once CC advertise doppler endpoint, and
	//everyone has migrated from loggregator to doppler
	c.read(func() {
		dopplerEndpoint = c.dopplerEndPoint
	})

	if dopplerEndpoint == "" {
		return strings.Replace(c.LoggregatorEndpoint(), "loggregator", "doppler", 1)
	}
	return
}

func (c *TerraformRepository) UaaEndpoint() (uaaEndpoint string) {
	c.read(func() {
		uaaEndpoint = c.uaaEndpoint
	})
	return
}

func (c *TerraformRepository) RoutingAPIEndpoint() (routingAPIEndpoint string) {
	c.read(func() {
		routingAPIEndpoint = c.routingAPIEndpoint
	})
	return
}

func (c *TerraformRepository) APIEndpoint() (apiEndpoint string) {
	c.read(func() {
		apiEndpoint = c.target
	})
	return
}

func (c *TerraformRepository) HasAPIEndpoint() (hasEndpoint bool) {
	c.read(func() {
		hasEndpoint = c.apiVersion != "" && c.target != ""
	})
	return
}

func (c *TerraformRepository) AccessToken() (accessToken string) {
	c.read(func() {
		accessToken = c.accessToken
	})
	return
}

func (c *TerraformRepository) SSHOAuthClient() (clientID string) {
	c.read(func() {
		clientID = c.sshOAuthClient
	})
	return
}

func (c *TerraformRepository) RefreshToken() (refreshToken string) {
	c.read(func() {
		refreshToken = c.refreshToken
	})
	return
}

func (c *TerraformRepository) OrganizationFields() (org models.OrganizationFields) {
	c.read(func() {
		org = models.OrganizationFields{}
	})
	return
}

func (c *TerraformRepository) SpaceFields() (space models.SpaceFields) {
	c.read(func() {
		space = models.SpaceFields{}
	})
	return
}

func (c *TerraformRepository) UserEmail() (email string) {
	c.read(func() {
		email = coreconfig.NewTokenInfo(c.accessToken).Email
	})
	return
}

func (c *TerraformRepository) UserGUID() (guid string) {
	c.read(func() {
		guid = coreconfig.NewTokenInfo(c.accessToken).UserGUID
	})
	return
}

func (c *TerraformRepository) Username() (name string) {
	c.read(func() {
		name = coreconfig.NewTokenInfo(c.accessToken).Username
	})
	return
}

func (c *TerraformRepository) IsLoggedIn() (loggedIn bool) {
	c.read(func() {
		loggedIn = c.accessToken != ""
	})
	return
}

func (c *TerraformRepository) HasOrganization() (hasOrg bool) {
	c.read(func() {
		hasOrg = false
	})
	return
}

func (c *TerraformRepository) HasSpace() (hasSpace bool) {
	c.read(func() {
		hasSpace = false
	})
	return
}

func (c *TerraformRepository) IsSSLDisabled() (isSSLDisabled bool) {
	c.read(func() {
		isSSLDisabled = c.sslDisabled
	})
	return
}

func (c *TerraformRepository) IsMinAPIVersion(requiredVersion semver.Version) bool {
	var apiVersion string
	c.read(func() {
		apiVersion = c.apiVersion
	})

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
	c.read(func() {
		minCLIVersion = c.minCLIVersion
	})
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

func (c *TerraformRepository) MinCLIVersion() (minCLIVersion string) {
	c.read(func() {
		minCLIVersion = c.minCLIVersion
	})
	return
}

func (c *TerraformRepository) MinRecommendedCLIVersion() (minRecommendedCLIVersion string) {
	c.read(func() {
		minRecommendedCLIVersion = c.minRecommendedCLIVersion
	})
	return
}

func (c *TerraformRepository) AsyncTimeout() (timeout uint) {
	c.read(func() {
		timeout = c.asyncTimeout
	})
	return
}

func (c *TerraformRepository) Trace() (trace string) {
	c.read(func() {
		trace = c.trace
	})
	return
}

func (c *TerraformRepository) ColorEnabled() (enabled string) {
	c.read(func() {
		enabled = c.colorEnabled
	})
	return
}

func (c *TerraformRepository) Locale() (locale string) {
	c.read(func() {
		locale = c.locale
	})
	return
}
func (c *TerraformRepository) PluginRepos() ([]models.PluginRepo) {
	return make([]models.PluginRepo, 0)
}

// SETTERS

func (c *TerraformRepository) ClearSession() {
	c.write(func() {
		c.accessToken = ""
		c.refreshToken = ""
	})
}

func (c *TerraformRepository) SetAPIEndpoint(endpoint string) {
	c.write(func() {
		c.target = endpoint
	})
}

func (c *TerraformRepository) SetAPIVersion(version string) {
	c.write(func() {
		c.apiVersion = version
	})
}

func (c *TerraformRepository) SetMinCLIVersion(version string) {
	c.write(func() {
		c.minCLIVersion = version
	})
}

func (c *TerraformRepository) SetMinRecommendedCLIVersion(version string) {
	c.write(func() {
		c.minRecommendedCLIVersion = version
	})
}

func (c *TerraformRepository) SetAuthenticationEndpoint(endpoint string) {
	c.write(func() {
		c.authorizationEndpoint = endpoint
	})
}

func (c *TerraformRepository) SetLoggregatorEndpoint(endpoint string) {
	c.write(func() {
		c.loggregatorEndPoint = endpoint
	})
}

func (c *TerraformRepository) SetDopplerEndpoint(endpoint string) {
	c.write(func() {
		c.dopplerEndPoint = endpoint
	})
}

func (c *TerraformRepository) SetUaaEndpoint(uaaEndpoint string) {
	c.write(func() {
		c.uaaEndpoint = uaaEndpoint
	})
}

func (c *TerraformRepository) SetRoutingAPIEndpoint(routingAPIEndpoint string) {
	c.write(func() {
		c.routingAPIEndpoint = routingAPIEndpoint
	})
}

func (c *TerraformRepository) SetAccessToken(token string) {
	c.write(func() {
		c.accessToken = token
	})
}

func (c *TerraformRepository) SetSSHOAuthClient(clientID string) {
	c.write(func() {
		c.sshOAuthClient = clientID
	})
}

func (c *TerraformRepository) SetRefreshToken(token string) {
	c.write(func() {
		c.refreshToken = token
	})
}

func (c *TerraformRepository) SetOrganizationFields(org models.OrganizationFields) {

}

func (c *TerraformRepository) SetSpaceFields(space models.SpaceFields) {

}

func (c *TerraformRepository) SetSSLDisabled(disabled bool) {
	c.write(func() {
		c.sslDisabled = disabled
	})
}

func (c *TerraformRepository) SetAsyncTimeout(timeout uint) {
	c.write(func() {
		c.asyncTimeout = timeout
	})
}

func (c *TerraformRepository) SetTrace(value string) {
	c.write(func() {
		c.trace = value
	})
}

func (c *TerraformRepository) SetColorEnabled(enabled string) {
	c.write(func() {
		c.colorEnabled = enabled
	})
}

func (c *TerraformRepository) SetLocale(locale string) {
	c.write(func() {
		c.locale = locale
	})
}

func (c *TerraformRepository) SetPluginRepo(repo models.PluginRepo) {
}

func (c *TerraformRepository) UnSetPluginRepo(index int) {

}
func (c *TerraformRepository) read(cb func()) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	cb()
}

func (c *TerraformRepository) write(cb func()) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	cb()
}

// CLOSERS

func (c *TerraformRepository) Close() {
	c.read(func() {
		// perform a read to ensure write lock has been cleared
	})
}
func NewTerraformRepository() coreconfig.Repository {
	return &TerraformRepository{
		mutex: new(sync.RWMutex),
	}
}
