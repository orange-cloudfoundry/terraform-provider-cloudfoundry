package resources

import "github.com/hashicorp/terraform/helper/schema"

type CfResource interface {
	Create(*schema.ResourceData, interface{}) error
	Read(*schema.ResourceData, interface{}) error
	Update(*schema.ResourceData, interface{}) error
	Delete(*schema.ResourceData, interface{}) error
	Exists(*schema.ResourceData, interface{}) (bool, error)
	Schema() map[string]*schema.Schema
}

const (
	ORG_RESOURCE = "organization"
	SPACE_RESOURCE = "space"
	QUOTA_RESOURCE = "quota"
	SEC_GROUP_RESOURCE = "sec_group"
	BUILDPACK_RESOURCE = "buildpack"
	SERVICE_BROKER_RESOURCE = "service_broker"
)

var resourceToLoad []string = []string{
	ORG_RESOURCE,
	SPACE_RESOURCE,
	QUOTA_RESOURCE,
	SEC_GROUP_RESOURCE,
	BUILDPACK_RESOURCE,
	SERVICE_BROKER_RESOURCE,
}

func RetrieveResourceMap() map[string]*schema.Resource {
	resources := make(map[string]*schema.Resource)
	for _, resourceType := range resourceToLoad {
		resources["cloudfoundry_" + resourceType] = FactoryCfResource(resourceType)
	}
	return resources
}
func FactoryCfResource(resourceType string) *schema.Resource {
	switch resourceType {
	case ORG_RESOURCE:
		return loadCfResource(NewCfOrganizationResource())
	case SPACE_RESOURCE:
		return loadCfResource(NewCfSpaceResource())
	case QUOTA_RESOURCE:
		return loadCfResource(NewCfQuotaResource())
	case SEC_GROUP_RESOURCE:
		return loadCfResource(NewCfSecurityGroupResource())
	case BUILDPACK_RESOURCE:
		return loadCfResource(NewCfBuildpackResource())
	case SERVICE_BROKER_RESOURCE:
		return loadCfResource(NewCfServiceBrokerResource())
	}
	return nil
}
func loadCfResource(cfResource CfResource) *schema.Resource {
	return &schema.Resource{
		Create: cfResource.Create,
		Read:   cfResource.Read,
		Update: cfResource.Update,
		Delete: cfResource.Delete,
		Exists: cfResource.Exists,
		Schema: cfResource.Schema(),
	}
}