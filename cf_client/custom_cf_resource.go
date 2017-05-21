package cf_client

import (
	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/models"
)

type PaginatedServiceInstanceResources struct {
	TotalResults int `json:"total_results"`
	Resources    []ServiceInstanceResource
}

type ServiceInstanceResource struct {
	resources.Resource
	Entity ServiceInstanceEntity
}

type LastOperation struct {
	Type        string `json:"type"`
	State       string `json:"state"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ServiceInstanceEntity struct {
	Name            string                             `json:"name"`
	DashboardURL    string                             `json:"dashboard_url"`
	SysLogDrainURL  string                             `json:"syslog_drain_url"`
	RouteServiceURL string                             `json:"route_service_url"`
	Tags            []string                           `json:"tags"`
	ServiceBindings []resources.ServiceBindingResource `json:"service_bindings"`
	ServiceKeys     []resources.ServiceKeyResource     `json:"service_keys"`
	ServicePlan     resources.ServicePlanResource      `json:"service_plan"`
	LastOperation   LastOperation                      `json:"last_operation"`
}

func (resource ServiceInstanceResource) ToFields() models.ServiceInstanceFields {
	return models.ServiceInstanceFields{
		GUID:            resource.Metadata.GUID,
		Name:            resource.Entity.Name,
		Tags:            resource.Entity.Tags,
		DashboardURL:    resource.Entity.DashboardURL,
		SysLogDrainURL:  resource.Entity.SysLogDrainURL,
		RouteServiceURL: resource.Entity.RouteServiceURL,
		LastOperation: models.LastOperationFields{
			Type:        resource.Entity.LastOperation.Type,
			State:       resource.Entity.LastOperation.State,
			Description: resource.Entity.LastOperation.Description,
			CreatedAt:   resource.Entity.LastOperation.CreatedAt,
			UpdatedAt:   resource.Entity.LastOperation.UpdatedAt,
		},
	}
}

func (resource ServiceInstanceResource) ToModel() (instance models.ServiceInstance) {
	instance.ServiceInstanceFields = resource.ToFields()
	instance.ServicePlan = resource.Entity.ServicePlan.ToFields()

	instance.ServiceBindings = []models.ServiceBindingFields{}
	for _, bindingResource := range resource.Entity.ServiceBindings {
		instance.ServiceBindings = append(instance.ServiceBindings, bindingResource.ToFields())
	}

	instance.ServiceKeys = []models.ServiceKeyFields{}
	for _, keyResource := range resource.Entity.ServiceKeys {
		instance.ServiceKeys = append(instance.ServiceKeys, keyResource.ToFields())
	}
	return
}
