package resources

import (
	"code.cloudfoundry.org/cli/cf/models"
	"strings"
)

func GetMissingSecGroup(sliceSource, sliceToInspect []models.SecurityGroupFields) []models.SecurityGroupFields {
	elementsNotFound := make([]models.SecurityGroupFields, 0)
	for _, elt := range sliceSource {
		if !containsSecGroup(sliceToInspect, elt) {
			elementsNotFound = append(elementsNotFound, elt)
		}
	}
	return elementsNotFound
}

func containsSecGroup(s []models.SecurityGroupFields, e models.SecurityGroupFields) bool {
	for _, a := range s {
		if a.GUID == e.GUID {
			return true
		}
	}
	return false
}
func IsWebURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}
