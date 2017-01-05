package cf_client

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2"
	"code.cloudfoundry.org/cli/cf/trace"
	"code.cloudfoundry.org/cli/cf/i18n"
	"code.cloudfoundry.org/cli/cf/api/authentication"
	"code.cloudfoundry.org/cli/cf/net"
	"io/ioutil"
	"code.cloudfoundry.org/cli/cf/api/organizations"
	"code.cloudfoundry.org/cli/cf/api/spaces"
	"code.cloudfoundry.org/cli/cf/api/securitygroups"
	spacesbinder "code.cloudfoundry.org/cli/cf/api/securitygroups/spaces"
	secgrouprun "code.cloudfoundry.org/cli/cf/api/securitygroups/defaults/running"
	secgroupstag "code.cloudfoundry.org/cli/cf/api/securitygroups/defaults/staging"
	"code.cloudfoundry.org/cli/cf/api"
	"code.cloudfoundry.org/cli/cf/api/spacequotas"
	"code.cloudfoundry.org/cli/cf/api/quotas"
	"code.cloudfoundry.org/cli/cf/appfiles"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/encryption"
)

type Client interface {
	Organizations() organizations.OrganizationRepository
	Spaces() spaces.SpaceRepository
	SecurityGroups() securitygroups.SecurityGroupRepo
	SecurityGroupsSpaceBinder() spacesbinder.SecurityGroupSpaceBinder
	SecurityGroupsRunningBinder() secgrouprun.SecurityGroupsRepo
	SecurityGroupsStagingBinder() secgroupstag.SecurityGroupsRepo
	ServiceBrokers() api.ServiceBrokerRepository
	ServicePlanVisibilities() api.ServicePlanVisibilityRepository
	ServicePlans() api.ServicePlanRepository
	Services() api.ServiceRepository
	SpaceQuotas() spacequotas.SpaceQuotaRepository
	Quotas() quotas.QuotaRepository
	Config() Config
	Buildpack() api.BuildpackRepository
	BuildpackBits() api.BuildpackBitsRepository
	Decrypter() encryption.Decrypter
}
type CfClient struct {
	config                      Config
	gateways                    CloudFoundryGateways
	organizations               organizations.OrganizationRepository
	spaces                      spaces.SpaceRepository
	securityGroups              securitygroups.SecurityGroupRepo
	serviceBrokers              api.ServiceBrokerRepository
	servicePlanVisibilities     api.ServicePlanVisibilityRepository
	spaceQuotas                 spacequotas.SpaceQuotaRepository
	quotas                      quotas.QuotaRepository
	buildpack                   api.BuildpackRepository
	buildpackBits               api.BuildpackBitsRepository
	securityGroupsSpaceBinder   spacesbinder.SecurityGroupSpaceBinder
	securityGroupsRunningBinder secgrouprun.SecurityGroupsRepo
	securityGroupsStagingBinder secgroupstag.SecurityGroupsRepo
	servicePlans                api.ServicePlanRepository
	decrypter                   encryption.Decrypter
	services                    api.ServiceRepository
}

func NewCfClient(config Config) (Client, error) {
	cfClient := &CfClient{config: config}
	err := cfClient.Init()
	if err != nil {
		return nil, err
	}
	return cfClient, err
}
func (client *CfClient) Init() error {
	ccClient := ccv2.NewCloudControllerClient()
	_, err := ccClient.TargetCF(client.config.Target(), client.config.SkipSSLValidation())
	if err != nil {
		return err
	}
	repository := NewTerraformRepository()
	repository.SetAPIEndpoint(client.config.ApiEndpoint)
	repository.SetAPIVersion(ccClient.APIVersion())
	repository.SetAsyncTimeout(uint(30))
	repository.SetAuthenticationEndpoint(ccClient.AuthorizationEndpoint())
	repository.SetDopplerEndpoint(ccClient.DopplerEndpoint())
	repository.SetLoggregatorEndpoint(ccClient.LoggregatorEndpoint())
	repository.SetRoutingAPIEndpoint(ccClient.RoutingEndpoint())
	repository.SetSSLDisabled(client.config.SkipInsecureSSL)
	repository.SetSSHOAuthClient(client.config.ClientID())
	repository.SetAccessToken(client.config.AccessToken())
	repository.SetRefreshToken(client.config.RefreshToken())
	repository.SetUaaEndpoint(ccClient.TokenEndpoint())
	repository.SetLocale(client.config.Locale)
	i18n.T = i18n.Init(repository)
	//Retry Wrapper
	logger := NewCfLogger(client.config.Verbose)
	gateways := NewCloudFoundryGateways(
		repository,
		logger,
	)

	client.gateways = gateways
	client.Authenticate()
	client.LoadRepositories()
	client.LoadDecrypter()
	return nil
}
func (client *CfClient) Authenticate() error {
	if client.config.AccessToken() != "" {
		return nil
	}
	gateways := client.gateways
	repository := gateways.Config
	uaaRepo := authentication.NewUAARepository(gateways.UAAGateway,
		repository,
		net.NewRequestDumper(trace.NewLogger(ioutil.Discard, false, "", "")),
	)
	err := uaaRepo.Authenticate(map[string]string{"username": client.config.Username, "password": client.config.Password})
	if err != nil {
		return err
	}
	return nil
}
func (client *CfClient) LoadDecrypter() {
	client.decrypter = encryption.NewPgpDecrypter(client.config.EncPrivateKey, client.config.Passphrase)
}
func (client *CfClient) LoadRepositories() {
	gateways := client.gateways
	repository := gateways.Config
	client.organizations = organizations.NewCloudControllerOrganizationRepository(repository, gateways.CloudControllerGateway)
	client.spaces = spaces.NewCloudControllerSpaceRepository(repository, gateways.CloudControllerGateway)
	client.securityGroups = securitygroups.NewSecurityGroupRepo(repository, gateways.CloudControllerGateway)
	client.serviceBrokers = api.NewCloudControllerServiceBrokerRepository(repository, gateways.CloudControllerGateway)
	client.servicePlanVisibilities = api.NewCloudControllerServicePlanVisibilityRepository(repository, gateways.CloudControllerGateway)
	client.spaceQuotas = spacequotas.NewCloudControllerSpaceQuotaRepository(repository, gateways.CloudControllerGateway)
	client.quotas = quotas.NewCloudControllerQuotaRepository(repository, gateways.CloudControllerGateway)
	client.buildpack = api.NewCloudControllerBuildpackRepository(repository, gateways.CloudControllerGateway)
	client.buildpackBits = api.NewCloudControllerBuildpackBitsRepository(repository, gateways.CloudControllerGateway, appfiles.ApplicationZipper{})
	client.securityGroupsSpaceBinder = spacesbinder.NewSecurityGroupSpaceBinder(repository, gateways.CloudControllerGateway)
	client.securityGroupsRunningBinder = secgrouprun.NewSecurityGroupsRepo(repository, gateways.CloudControllerGateway)
	client.securityGroupsStagingBinder = secgroupstag.NewSecurityGroupsRepo(repository, gateways.CloudControllerGateway)
	client.servicePlans = api.NewCloudControllerServicePlanRepository(repository, gateways.CloudControllerGateway)
	client.services = api.NewCloudControllerServiceRepository(repository, gateways.CloudControllerGateway)
}
func (client CfClient) Organizations() organizations.OrganizationRepository {
	return client.organizations
}

func (client CfClient) Spaces() spaces.SpaceRepository {
	return client.spaces
}
func (client CfClient) SecurityGroups() securitygroups.SecurityGroupRepo {
	return client.securityGroups
}
func (client CfClient) ServiceBrokers() api.ServiceBrokerRepository {
	return client.serviceBrokers
}
func (client CfClient) ServicePlanVisibilities() api.ServicePlanVisibilityRepository {
	return client.servicePlanVisibilities
}
func (client CfClient) SpaceQuotas() spacequotas.SpaceQuotaRepository {
	return client.spaceQuotas
}

func (client CfClient) Quotas() quotas.QuotaRepository {
	return client.quotas
}

func (client CfClient) Config() Config {
	return client.config
}
func (client CfClient) Buildpack() api.BuildpackRepository {
	return client.buildpack
}
func (client CfClient) BuildpackBits() api.BuildpackBitsRepository {
	return client.buildpackBits
}
func (client CfClient) SecurityGroupsSpaceBinder() spacesbinder.SecurityGroupSpaceBinder {
	return client.securityGroupsSpaceBinder
}
func (client CfClient) SecurityGroupsRunningBinder() secgrouprun.SecurityGroupsRepo {
	return client.securityGroupsRunningBinder
}
func (client CfClient) SecurityGroupsStagingBinder() secgroupstag.SecurityGroupsRepo {
	return client.securityGroupsStagingBinder
}
func (client CfClient) ServicePlans() api.ServicePlanRepository {
	return client.servicePlans
}
func (client CfClient) Services() api.ServiceRepository {
	return client.services
}
func (client CfClient) Decrypter() encryption.Decrypter {
	return client.decrypter
}