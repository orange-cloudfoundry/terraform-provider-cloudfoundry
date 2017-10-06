package cf_client

import (
	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
	"fmt"
)

type FinderRepository interface {
	GetDomainFromCf(domain models.DomainFields) (models.DomainFields, error)
	GetBuildpackFromCf(bpGuid string) (models.Buildpack, error)
	GetQuotaFromCf(quotaGuid string, isOrgQuota bool) (interface{}, error)
	GetRouteFromCf(routeGuid string) (models.Route, error)
	GetSecGroupFromCf(secGroupId string) (models.SecurityGroup, error)
	GetServiceFromCf(svcGuid string) (models.ServiceInstance, error)
	GetSpaceFromCf(spaceGuid string) (models.Space, error)
	GetAppFromCf(appGuid string) (models.Application, error)
	GetServiceBindingsFromApp(appGuid string) ([]ServiceBindingFields, error)
}

type Finder struct {
	config    Config
	ccGateway net.Gateway
}

func NewFinderRepository(config Config, ccGateway net.Gateway) FinderRepository {
	return &Finder{
		config:    config,
		ccGateway: ccGateway,
	}
}
func (f Finder) GetDomainFromCf(domain models.DomainFields) (models.DomainFields, error) {
	res := resources.DomainResource{}
	err := f.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/private_domains/%s?inline-relations-depth=1",
			f.config.ApiEndpoint,
			domain.GUID,
		),
		&res)
	if err != nil {
		err = f.ccGateway.GetResource(
			fmt.Sprintf("%s/v2/shared_domains/%s?inline-relations-depth=1",
				f.config.ApiEndpoint,
				domain.GUID,
			),
			&res)
	}
	if err != nil {
		if _, ok := err.(*errors.HTTPNotFoundError); ok {
			return models.DomainFields{}, nil
		}
		return models.DomainFields{}, err
	}
	return res.ToFields(), nil
}
func (f Finder) GetBuildpackFromCf(bpGuid string) (models.Buildpack, error) {
	res := resources.BuildpackResource{}
	err := f.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/buildpacks/%s?inline-relations-depth=1", f.config.ApiEndpoint, bpGuid),
		&res)
	if err != nil {
		if _, ok := err.(*errors.HTTPNotFoundError); ok {
			return models.Buildpack{}, nil
		}
		return models.Buildpack{}, err
	}
	return res.ToFields(), nil
}
func (f Finder) GetQuotaFromCf(quotaGuid string, isOrgQuota bool) (interface{}, error) {
	var err error

	if isOrgQuota {
		res := resources.QuotaResource{}
		err = f.ccGateway.GetResource(
			fmt.Sprintf("%s/v2/quota_definitions/%s?inline-relations-depth=1", f.config.ApiEndpoint, quotaGuid),
			&res)
		if err != nil {
			if _, ok := err.(*errors.HTTPNotFoundError); ok {
				return models.QuotaFields{}, nil
			}
			return models.QuotaFields{}, err
		}
		return res.ToFields(), nil
	}
	res := resources.SpaceQuotaResource{}
	err = f.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/space_quota_definitions/%s?inline-relations-depth=1", f.config.ApiEndpoint, quotaGuid),
		&res)
	if err != nil {
		if _, ok := err.(*errors.HTTPNotFoundError); ok {
			return models.QuotaFields{}, nil
		}
		return models.QuotaFields{}, err
	}
	return res.ToModel(), nil
}
func (f Finder) GetRouteFromCf(routeGuid string) (models.Route, error) {
	routeRes := resources.RouteResource{}
	err := f.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/routes/%s?inline-relations-depth=1", f.config.ApiEndpoint, routeGuid),
		&routeRes)
	if err != nil {
		if _, ok := err.(*errors.HTTPNotFoundError); ok {
			return models.Route{}, nil
		}
		return models.Route{}, err
	}
	return routeRes.ToModel(), nil
}
func (f Finder) GetSecGroupFromCf(secGroupId string) (models.SecurityGroup, error) {
	res := resources.SecurityGroupResource{}
	err := f.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/security_groups/%s?inline-relations-depth=1", f.config.ApiEndpoint, secGroupId),
		&res)
	if err != nil {
		if _, ok := err.(*errors.HTTPNotFoundError); ok {
			return models.SecurityGroup{}, nil
		}
		return models.SecurityGroup{}, err
	}
	secGroup := res.ToModel()
	err = f.ccGateway.ListPaginatedResources(
		f.config.ApiEndpoint,
		secGroup.SpaceURL+"?inline-relations-depth=1",
		resources.SpaceResource{},
		func(resource interface{}) bool {
			if asgr, ok := resource.(resources.SpaceResource); ok {
				secGroup.Spaces = append(secGroup.Spaces, asgr.ToModel())
				return true
			}
			return false
		},
	)
	return secGroup, nil
}
func (f Finder) GetServiceFromCf(svcGuid string) (models.ServiceInstance, error) {
	res := ServiceInstanceResource{}
	err := f.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/service_instances/%s?inline-relations-depth=1", f.config.ApiEndpoint, svcGuid),
		&res)
	if err != nil {
		if _, ok := err.(*errors.HTTPNotFoundError); ok {
			return models.ServiceInstance{}, nil
		}
		return models.ServiceInstance{}, err
	}
	model := res.ToModel()

	return model, nil
}
func (f Finder) GetSpaceFromCf(spaceGuid string) (models.Space, error) {
	res := resources.SpaceResource{}
	err := f.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/spaces/%s?inline-relations-depth=1", f.config.ApiEndpoint, spaceGuid),
		&res)
	if _, ok := err.(*errors.HTTPNotFoundError); ok {
		return models.Space{}, nil
	}
	if err != nil {
		return models.Space{}, err
	}
	return res.ToModel(), nil
}
func (f Finder) GetServiceBindingsFromApp(appGuid string) ([]ServiceBindingFields, error) {
	serviceBindings := []ServiceBindingFields{}
	err := f.ccGateway.ListPaginatedResources(
		f.config.ApiEndpoint,
		fmt.Sprintf("/v2/apps/%s/service_bindings", appGuid),
		ServiceBindingResource{},
		func(resource interface{}) bool {
			if serviceBindingResource, ok := resource.(ServiceBindingResource); ok {
				serviceBindings = append(serviceBindings, serviceBindingResource.ToFields())
			}
			return true
		},
	)
	return serviceBindings, err

}
func (f Finder) GetAppFromCf(appGuid string) (models.Application, error) {
	res := resources.ApplicationResource{}
	err := f.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/apps/%s?inline-relations-depth=1", f.config.ApiEndpoint, appGuid),
		&res)
	if _, ok := err.(*errors.HTTPNotFoundError); ok {
		return models.Application{}, nil
	}
	if err != nil {
		return models.Application{}, err
	}
	model := res.ToModel()
	if res.Entity.HealthCheckTimeout != nil {
		model.HealthCheckTimeout = *res.Entity.HealthCheckTimeout
	}
	return model, nil
}
