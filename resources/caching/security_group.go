package caching

import (
	"code.cloudfoundry.org/cli/cf/models"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
)

var secGroups []models.SecurityGroup

func GetSecGroupsFromCf(client cf_client.Client) ([]models.SecurityGroup, error) {
	var err error
	if secGroups != nil {
		return secGroups, nil
	}
	secGroups, err = client.SecurityGroups().FindAll()
	if err != nil {
		return make([]models.SecurityGroup, 0), err
	}
	return secGroups, err
}

func GetSecGroupFromCf(client cf_client.Client, secGroupId string) (models.SecurityGroup, error) {
	secGroups, err := GetSecGroupsFromCf(client)
	if err != nil {
		return models.SecurityGroup{}, err
	}
	for _, secGroup := range secGroups {
		if secGroup.GUID == secGroupId {
			return secGroup, nil
		}
	}
	return models.SecurityGroup{}, nil
}
