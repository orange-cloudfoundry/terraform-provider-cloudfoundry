package caching

import (
	"code.cloudfoundry.org/cli/cf/models"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
)

var quotas []models.QuotaFields
var spaceQuotas []models.SpaceQuota

func GetQuotas(client cf_client.Client, update bool) ([]models.QuotaFields, error) {
	var err error
	if quotas != nil && !update {
		return quotas, nil
	}
	quotas, err = client.Quotas().FindAll()
	if err != nil {
		return make([]models.QuotaFields, 0), err
	}
	return quotas, err
}

func GetSpaceQuotasFromOrg(client cf_client.Client, orgId string, update bool) ([]models.SpaceQuota, error) {
	var err error
	if spaceQuotas != nil && !update {
		return spaceQuotas, nil
	}
	spaceQuotas, err = client.SpaceQuotas().FindByOrg(orgId)
	if err != nil {
		return make([]models.SpaceQuota, 0), err
	}
	return spaceQuotas, err
}
