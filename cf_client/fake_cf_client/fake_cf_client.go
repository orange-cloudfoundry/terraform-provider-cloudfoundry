package fake_cf_client

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/cf/api"
	"code.cloudfoundry.org/cli/cf/api/apifakes"
	"code.cloudfoundry.org/cli/cf/api/appinstances"
	"code.cloudfoundry.org/cli/cf/api/applications"
	"code.cloudfoundry.org/cli/cf/api/environmentvariablegroups"
	"code.cloudfoundry.org/cli/cf/api/featureflags"
	"code.cloudfoundry.org/cli/cf/api/logs"
	"code.cloudfoundry.org/cli/cf/api/organizations"
	"code.cloudfoundry.org/cli/cf/api/organizations/organizationsfakes"
	"code.cloudfoundry.org/cli/cf/api/quotas"
	"code.cloudfoundry.org/cli/cf/api/quotas/quotasfakes"
	"code.cloudfoundry.org/cli/cf/api/securitygroups"
	secgrouprun "code.cloudfoundry.org/cli/cf/api/securitygroups/defaults/running"
	secgrouprunfake "code.cloudfoundry.org/cli/cf/api/securitygroups/defaults/running/runningfakes"
	secgroupstag "code.cloudfoundry.org/cli/cf/api/securitygroups/defaults/staging"
	secgroupstagfake "code.cloudfoundry.org/cli/cf/api/securitygroups/defaults/staging/stagingfakes"
	"code.cloudfoundry.org/cli/cf/api/securitygroups/securitygroupsfakes"
	spacesbinder "code.cloudfoundry.org/cli/cf/api/securitygroups/spaces"
	spacesbinderfake "code.cloudfoundry.org/cli/cf/api/securitygroups/spaces/spacesfakes"
	"code.cloudfoundry.org/cli/cf/api/spacequotas"
	"code.cloudfoundry.org/cli/cf/api/spacequotas/spacequotasfakes"
	"code.cloudfoundry.org/cli/cf/api/spaces"
	"code.cloudfoundry.org/cli/cf/api/spaces/spacesfakes"
	"code.cloudfoundry.org/cli/cf/api/stacks"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/bitsmanager"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/bitsmanager/bitsmanagerfakes"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/encryption"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/encryption/fake_encryption"
)

type FakeCfClient struct {
	config                      cf_client.Config
	organizations               *organizationsfakes.FakeOrganizationRepository
	spaces                      *spacesfakes.FakeSpaceRepository
	securityGroups              *securitygroupsfakes.FakeSecurityGroupRepo
	serviceBrokers              *apifakes.FakeServiceBrokerRepository
	servicePlanVisibilities     *apifakes.FakeServicePlanVisibilityRepository
	spaceQuotas                 *spacequotasfakes.FakeSpaceQuotaRepository
	quotas                      *quotasfakes.FakeQuotaRepository
	buildpack                   *apifakes.FakeBuildpackRepository
	buildpackBits               *apifakes.FakeBuildpackBitsRepository
	securityGroupsSpaceBinder   *spacesbinderfake.FakeSecurityGroupSpaceBinder
	securityGroupsRunningBinder *secgrouprunfake.FakeSecurityGroupsRepo
	securityGroupsStagingBinder *secgroupstagfake.FakeSecurityGroupsRepo
	servicePlans                *apifakes.FakeServicePlanRepository
	decrypter                   encryption.Decrypter
	services                    *apifakes.FakeServiceRepository
	serviceBinding              *apifakes.FakeServiceBindingRepository
	domain                      *apifakes.FakeDomainRepository
	routingApi                  *apifakes.FakeRoutingAPIRepository
	route                       *apifakes.FakeRouteRepository
	routeServiceBinding         *apifakes.FakeRouteServiceBindingRepository
	userProvidedService         *apifakes.FakeUserProvidedServiceInstanceRepository
	finder                      *FakeFinderRepository
	applicationBits             *bitsmanagerfakes.FakeApplicationBitsRepository
}

func NewFakeCfClient() *FakeCfClient {
	client := &FakeCfClient{}
	client.Init()
	return client
}
func (c *FakeCfClient) GetClient() cf_client.Client {
	return c
}
func (c *FakeCfClient) Init() {
	c.config = cf_client.Config{
		ApiEndpoint: "http://fake.api.endpoint.com",
	}
	c.organizations = new(organizationsfakes.FakeOrganizationRepository)
	c.spaces = new(spacesfakes.FakeSpaceRepository)
	c.securityGroups = new(securitygroupsfakes.FakeSecurityGroupRepo)
	c.serviceBrokers = new(apifakes.FakeServiceBrokerRepository)
	c.servicePlanVisibilities = new(apifakes.FakeServicePlanVisibilityRepository)
	c.spaceQuotas = new(spacequotasfakes.FakeSpaceQuotaRepository)
	c.quotas = new(quotasfakes.FakeQuotaRepository)
	c.buildpack = new(apifakes.FakeBuildpackRepository)
	c.buildpackBits = new(apifakes.FakeBuildpackBitsRepository)
	c.securityGroupsSpaceBinder = new(spacesbinderfake.FakeSecurityGroupSpaceBinder)
	c.securityGroupsRunningBinder = new(secgrouprunfake.FakeSecurityGroupsRepo)
	c.securityGroupsStagingBinder = new(secgroupstagfake.FakeSecurityGroupsRepo)
	c.servicePlans = new(apifakes.FakeServicePlanRepository)
	c.services = new(apifakes.FakeServiceRepository)
	c.serviceBinding = new(apifakes.FakeServiceBindingRepository)
	c.domain = new(apifakes.FakeDomainRepository)
	c.routingApi = new(apifakes.FakeRoutingAPIRepository)
	c.route = new(apifakes.FakeRouteRepository)
	c.routeServiceBinding = new(apifakes.FakeRouteServiceBindingRepository)
	c.userProvidedService = new(apifakes.FakeUserProvidedServiceInstanceRepository)
	c.applicationBits = new(bitsmanagerfakes.FakeApplicationBitsRepository)
	c.finder = new(FakeFinderRepository)
	c.decrypter = fake_encryption.NewFakeDecrypter()
}
func (client FakeCfClient) Organizations() organizations.OrganizationRepository {
	return client.organizations
}

func (client FakeCfClient) Spaces() spaces.SpaceRepository {
	return client.spaces
}
func (client FakeCfClient) SecurityGroups() securitygroups.SecurityGroupRepo {
	return client.securityGroups
}
func (client FakeCfClient) ServiceBrokers() api.ServiceBrokerRepository {
	return client.serviceBrokers
}
func (client FakeCfClient) ServicePlanVisibilities() api.ServicePlanVisibilityRepository {
	return client.servicePlanVisibilities
}
func (client FakeCfClient) SpaceQuotas() spacequotas.SpaceQuotaRepository {
	return client.spaceQuotas
}

func (client FakeCfClient) Quotas() quotas.QuotaRepository {
	return client.quotas
}

func (client FakeCfClient) Config() cf_client.Config {
	return client.config
}
func (client FakeCfClient) Buildpack() api.BuildpackRepository {
	return client.buildpack
}
func (client FakeCfClient) BuildpackBits() api.BuildpackBitsRepository {
	return client.buildpackBits
}
func (client FakeCfClient) SecurityGroupsSpaceBinder() spacesbinder.SecurityGroupSpaceBinder {
	return client.securityGroupsSpaceBinder
}
func (client FakeCfClient) SecurityGroupsRunningBinder() secgrouprun.SecurityGroupsRepo {
	return client.securityGroupsRunningBinder
}
func (client FakeCfClient) SecurityGroupsStagingBinder() secgroupstag.SecurityGroupsRepo {
	return client.securityGroupsStagingBinder
}
func (client FakeCfClient) ServicePlans() api.ServicePlanRepository {
	return client.servicePlans
}
func (client FakeCfClient) Services() api.ServiceRepository {
	return client.services
}
func (client FakeCfClient) ServiceBinding() api.ServiceBindingRepository {
	return client.serviceBinding
}
func (client FakeCfClient) Decrypter() encryption.Decrypter {
	return client.decrypter
}
func (client FakeCfClient) Domain() api.DomainRepository {
	return client.domain
}
func (client FakeCfClient) RoutingAPI() api.RoutingAPIRepository {
	return client.routingApi
}
func (client FakeCfClient) Route() api.RouteRepository {
	return client.route
}
func (client FakeCfClient) Gateways() cf_client.CloudFoundryGateways {
	return cf_client.CloudFoundryGateways{}
}
func (client FakeCfClient) Stack() stacks.CloudControllerStackRepository {
	return stacks.CloudControllerStackRepository{}
}

func (client FakeCfClient) RouteServiceBinding() api.RouteServiceBindingRepository {
	return client.routeServiceBinding
}
func (client FakeCfClient) UserProvidedService() api.UserProvidedServiceInstanceRepository {
	return client.userProvidedService
}
func (client FakeCfClient) Finder() cf_client.FinderRepository {
	return client.finder
}
func (client FakeCfClient) FeatureFlags() featureflags.FeatureFlagRepository {
	return &featureflags.CloudControllerFeatureFlagRepository{}
}
func (client FakeCfClient) EnvVarGroup() environmentvariablegroups.Repository {
	return environmentvariablegroups.CloudControllerRepository{}
}
func (client FakeCfClient) CCv3Client() *ccv3.Client {
	return &ccv3.Client{}
}
func (client FakeCfClient) Applications() applications.Repository {
	return applications.CloudControllerRepository{}
}
func (client FakeCfClient) AppInstances() appinstances.Repository {
	return appinstances.CloudControllerAppInstancesRepository{}
}
func (client FakeCfClient) ApplicationBits() bitsmanager.ApplicationBitsRepository {
	return client.applicationBits
}
func (client FakeCfClient) Logs() logs.Repository {
	return &logs.NoaaLogsRepository{}
}

// get Fake call -------

func (client FakeCfClient) FakeOrganizations() *organizationsfakes.FakeOrganizationRepository {
	return client.organizations
}

func (client FakeCfClient) FakeSpaces() *spacesfakes.FakeSpaceRepository {
	return client.spaces
}
func (client FakeCfClient) FakeSecurityGroups() *securitygroupsfakes.FakeSecurityGroupRepo {
	return client.securityGroups
}
func (client FakeCfClient) FakeServiceBrokers() *apifakes.FakeServiceBrokerRepository {
	return client.serviceBrokers
}
func (client FakeCfClient) FakeServicePlanVisibilities() *apifakes.FakeServicePlanVisibilityRepository {
	return client.servicePlanVisibilities
}
func (client FakeCfClient) FakeSpaceQuotas() *spacequotasfakes.FakeSpaceQuotaRepository {
	return client.spaceQuotas
}

func (client FakeCfClient) FakeQuotas() *quotasfakes.FakeQuotaRepository {
	return client.quotas
}

func (client FakeCfClient) FakeBuildpack() *apifakes.FakeBuildpackRepository {
	return client.buildpack
}
func (client FakeCfClient) FakeBuildpackBits() *apifakes.FakeBuildpackBitsRepository {
	return client.buildpackBits
}
func (client FakeCfClient) FakeSecurityGroupsSpaceBinder() *spacesbinderfake.FakeSecurityGroupSpaceBinder {
	return client.securityGroupsSpaceBinder
}
func (client FakeCfClient) FakeSecurityGroupsRunningBinder() *secgrouprunfake.FakeSecurityGroupsRepo {
	return client.securityGroupsRunningBinder
}
func (client FakeCfClient) FakeSecurityGroupsStagingBinder() *secgroupstagfake.FakeSecurityGroupsRepo {
	return client.securityGroupsStagingBinder
}
func (client FakeCfClient) FakeServicePlans() *apifakes.FakeServicePlanRepository {
	return client.servicePlans
}
func (client FakeCfClient) FakeServices() *apifakes.FakeServiceRepository {
	return client.services
}
func (client FakeCfClient) FakeServiceBinding() *apifakes.FakeServiceBindingRepository {
	return client.serviceBinding
}
func (client FakeCfClient) FakeDomain() api.DomainRepository {
	return client.domain
}
func (client FakeCfClient) FakeRoutingAPI() api.RoutingAPIRepository {
	return client.routingApi
}
func (client FakeCfClient) FakeRoute() api.RouteRepository {
	return client.route
}
func (client FakeCfClient) FakeRouteServiceBinding() *apifakes.FakeRouteServiceBindingRepository {
	return client.routeServiceBinding
}
func (client FakeCfClient) FakeUserProvidedService() *apifakes.FakeUserProvidedServiceInstanceRepository {
	return client.userProvidedService
}
func (client FakeCfClient) FakeFinder() *FakeFinderRepository {
	return client.finder
}
func (client FakeCfClient) FakeApplicationBits() *bitsmanagerfakes.FakeApplicationBitsRepository {
	return client.applicationBits
}
