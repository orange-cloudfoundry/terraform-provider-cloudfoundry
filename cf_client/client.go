package cf_client

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	ccWrapper "code.cloudfoundry.org/cli/api/cloudcontroller/wrapper"
	"code.cloudfoundry.org/cli/api/uaa"
	uaaWrapper "code.cloudfoundry.org/cli/api/uaa/wrapper"
	"code.cloudfoundry.org/cli/cf/api"
	"code.cloudfoundry.org/cli/cf/api/authentication"
	"code.cloudfoundry.org/cli/cf/api/environmentvariablegroups"
	"code.cloudfoundry.org/cli/cf/api/featureflags"
	"code.cloudfoundry.org/cli/cf/api/organizations"
	"code.cloudfoundry.org/cli/cf/api/quotas"
	"code.cloudfoundry.org/cli/cf/api/securitygroups"
	secgrouprun "code.cloudfoundry.org/cli/cf/api/securitygroups/defaults/running"
	secgroupstag "code.cloudfoundry.org/cli/cf/api/securitygroups/defaults/staging"
	spacesbinder "code.cloudfoundry.org/cli/cf/api/securitygroups/spaces"
	"code.cloudfoundry.org/cli/cf/api/spacequotas"
	"code.cloudfoundry.org/cli/cf/api/spaces"
	"code.cloudfoundry.org/cli/cf/api/stacks"
	"code.cloudfoundry.org/cli/cf/appfiles"
	"code.cloudfoundry.org/cli/cf/i18n"
	"code.cloudfoundry.org/cli/cf/net"
	"code.cloudfoundry.org/cli/cf/trace"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/encryption"
	"io/ioutil"
	"time"
)

type Client interface {
	Gateways() CloudFoundryGateways
	Finder() FinderRepository
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
	Domain() api.DomainRepository
	RoutingAPI() api.RoutingAPIRepository
	Route() api.RouteRepository
	Stack() stacks.CloudControllerStackRepository
	RouteServiceBinding() api.RouteServiceBindingRepository
	UserProvidedService() api.UserProvidedServiceInstanceRepository
	FeatureFlags() featureflags.FeatureFlagRepository
	EnvVarGroup() environmentvariablegroups.Repository
	CCv3Client() *ccv3.Client
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
	domain                      api.DomainRepository
	routingApi                  api.RoutingAPIRepository
	route                       api.RouteRepository
	stack                       stacks.CloudControllerStackRepository
	routeServiceBinding         api.RouteServiceBindingRepository
	userProvidedService         api.UserProvidedServiceInstanceRepository
	finder                      FinderRepository
	featureFlags                featureflags.FeatureFlagRepository
	envVarGroup                 environmentvariablegroups.Repository
	ccv3Client                  *ccv3.Client
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

	ccClient := ccv2.NewClient(ccv2.Config{
		AppName:            client.config.AppName,
		AppVersion:         client.config.AppVersion,
		JobPollingInterval: time.Duration(2) * time.Second,
		JobPollingTimeout:  time.Duration(10) * time.Second,
	})
	_, err := ccClient.TargetCF(ccv2.TargetSettings{
		DialTimeout:       time.Duration(1) * time.Second,
		URL:               client.config.Target(),
		SkipSSLValidation: client.config.SkipSSLValidation(),
	})
	if err != nil {
		return err
	}
	repository := NewTerraformRepository()
	repository.SetAPIEndpoint(client.config.ApiEndpoint)
	repository.SetAPIVersion(ccClient.APIVersion())
	repository.SetAsyncTimeout(uint(30))
	repository.SetAuthenticationEndpoint(ccClient.AuthorizationEndpoint())
	repository.SetDopplerEndpoint(ccClient.DopplerEndpoint())
	repository.SetRoutingAPIEndpoint(ccClient.RoutingEndpoint())
	repository.SetSSLDisabled(client.config.SkipInsecureSSL)
	repository.SetSSHOAuthClient(client.config.ClientID())
	repository.SetAccessToken(client.config.AccessToken())
	repository.SetRefreshToken(client.config.RefreshToken())
	repository.SetUaaEndpoint(ccClient.TokenEndpoint())
	repository.SetUAAOAuthClient("cf")
	repository.SetUAAOAuthClientSecret("")
	repository.SetLocale(client.config.Locale)
	i18n.T = i18n.Init(repository)
	//Retry Wrapper
	logger := NewCfLogger(client.config.Verbose)
	gateways := NewCloudFoundryGateways(
		repository,
		logger,
	)

	client.gateways = gateways
	err = client.Authenticate()
	if err != nil {
		return err
	}
	client.LoadRepositories()
	client.LoadDecrypter()
	client.LoadCCv3()
	return nil
}
func (client *CfClient) LoadCCv3() error {
	config := client.gateways.Config
	ccWrappers := []ccv3.ConnectionWrapper{}
	authWrapper := ccWrapper.NewUAAAuthentication(nil, config)
	ccWrappers = append(ccWrappers, authWrapper)
	ccWrappers = append(ccWrappers, ccWrapper.NewRetryRequest(2))

	ccClient := ccv3.NewClient(ccv3.Config{
		AppName:    client.config.AppName,
		AppVersion: client.config.AppVersion,
		Wrappers:   ccWrappers,
	})
	_, err := ccClient.TargetCF(ccv3.TargetSettings{
		DialTimeout:       time.Duration(1) * time.Second,
		URL:               client.config.Target(),
		SkipSSLValidation: client.config.SkipSSLValidation(),
	})
	if err != nil {
		return err
	}
	uaaClient := uaa.NewClient(uaa.Config{
		AppName:           client.config.AppName,
		AppVersion:        client.config.AppVersion,
		ClientID:          "cf",
		ClientSecret:      "",
		DialTimeout:       time.Duration(1) * time.Second,
		SkipSSLValidation: client.config.SkipSSLValidation(),
		URL:               ccClient.UAA(),
	})
	uaaClient.WrapConnection(uaaWrapper.NewUAAAuthentication(uaaClient, config))
	uaaClient.WrapConnection(uaaWrapper.NewRetryRequest(2))

	authWrapper.SetClient(uaaClient)
	client.ccv3Client = ccClient
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
		panic(err)
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
	client.finder = NewFinderRepository(client.config, gateways.CloudControllerGateway)
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
	client.domain = api.NewCloudControllerDomainRepository(repository, gateways.CloudControllerGateway)
	client.routingApi = api.NewRoutingAPIRepository(repository, gateways.CloudControllerGateway)
	client.route = api.NewCloudControllerRouteRepository(repository, gateways.CloudControllerGateway)
	client.stack = stacks.NewCloudControllerStackRepository(repository, gateways.CloudControllerGateway)
	client.routeServiceBinding = api.NewCloudControllerRouteServiceBindingRepository(repository, gateways.CloudControllerGateway)
	client.userProvidedService = api.NewCCUserProvidedServiceInstanceRepository(repository, gateways.CloudControllerGateway)
	client.featureFlags = featureflags.NewCloudControllerFeatureFlagRepository(repository, gateways.CloudControllerGateway)
	client.envVarGroup = environmentvariablegroups.NewCloudControllerRepository(repository, gateways.CloudControllerGateway)
}
func (client CfClient) Gateways() CloudFoundryGateways {
	return client.gateways
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
func (client CfClient) Domain() api.DomainRepository {
	return client.domain
}
func (client CfClient) RoutingAPI() api.RoutingAPIRepository {
	return client.routingApi
}
func (client CfClient) Route() api.RouteRepository {
	return client.route
}
func (client CfClient) Stack() stacks.CloudControllerStackRepository {
	return client.stack
}
func (client CfClient) RouteServiceBinding() api.RouteServiceBindingRepository {
	return client.routeServiceBinding
}
func (client CfClient) UserProvidedService() api.UserProvidedServiceInstanceRepository {
	return client.userProvidedService
}
func (client CfClient) Finder() FinderRepository {
	return client.finder
}
func (client CfClient) FeatureFlags() featureflags.FeatureFlagRepository {
	return client.featureFlags
}
func (client CfClient) EnvVarGroup() environmentvariablegroups.Repository {
	return client.envVarGroup
}
func (client CfClient) CCv3Client() *ccv3.Client {
	return client.ccv3Client
}
