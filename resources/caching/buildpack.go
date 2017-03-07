package caching

import (
	"code.cloudfoundry.org/cli/cf/models"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
)

var buildpacks []models.Buildpack

func GetBuildpacks(client cf_client.Client, update bool) ([]models.Buildpack, error) {
	var err error
	if buildpacks != nil && !update {
		return buildpacks, nil
	}
	buildpacks = make([]models.Buildpack, 0)
	err = client.Buildpack().ListBuildpacks(func(buildpack models.Buildpack) bool {
		buildpacks = append(buildpacks, buildpack)
		return true
	})
	if err != nil {
		return buildpacks, err
	}
	return buildpacks, err
}
