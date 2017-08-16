package cf_client

import (
	"code.cloudfoundry.org/cli/cf/api/resources"
)

type ServiceBindingResource struct {
	resources.Resource
	Entity ServiceBindingEntity
}

type ServiceBindingEntity struct {
	AppGUID             string `json:"app_guid"`
	ServiceInstanceGUID string `json:"service_instance_guid"`
}

type ServiceBindingFields struct {
	GUID                string
	URL                 string
	AppGUID             string
	ServiceInstanceGUID string
}

func (resource ServiceBindingResource) ToFields() ServiceBindingFields {
	return ServiceBindingFields{
		URL:                 resource.Metadata.URL,
		GUID:                resource.Metadata.GUID,
		AppGUID:             resource.Entity.AppGUID,
		ServiceInstanceGUID: resource.Entity.ServiceInstanceGUID,
	}
}
