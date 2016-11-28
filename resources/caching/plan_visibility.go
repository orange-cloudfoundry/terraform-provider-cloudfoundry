package caching

import (
	"code.cloudfoundry.org/cli/cf/models"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
)

var planVisibilities []models.ServicePlanVisibilityFields

func GetPlanVisibilities(client cf_client.Client, update bool) ([]models.ServicePlanVisibilityFields, error) {
	var err error
	if planVisibilities != nil && !update {
		return planVisibilities, nil
	}
	planVisibilities, err = client.ServicePlanVisibilities().List()
	if err != nil {
		return make([]models.ServicePlanVisibilityFields, 0), err
	}
	return planVisibilities, nil
}

func GetPlanVisibilitiesForPlan(client cf_client.Client, planId string, update bool) ([]models.ServicePlanVisibilityFields, error) {
	finalVisibilities := make([]models.ServicePlanVisibilityFields, 0)
	visibilities, err := GetPlanVisibilities(client, update)
	if err != nil {
		return finalVisibilities, err
	}

	for _, visibility := range visibilities {
		if visibility.ServicePlanGUID == planId {
			finalVisibilities = append(finalVisibilities, visibility)
		}
	}
	return finalVisibilities, nil
}
func GetPlanVisibilityForPlanAndOrg(client cf_client.Client, planId, orgId string, update bool) (models.ServicePlanVisibilityFields, error) {
	visibilities, err := GetPlanVisibilities(client, update)
	if err != nil {
		return models.ServicePlanVisibilityFields{}, err
	}
	for _, visibility := range visibilities {
		if visibility.ServicePlanGUID == planId && visibility.OrganizationGUID == orgId {
			return visibility, nil
		}
	}
	return models.ServicePlanVisibilityFields{}, nil
}
