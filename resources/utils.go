package resources

import (
	"code.cloudfoundry.org/cli/cf/models"
	"encoding/json"
	"strings"
)

// Giving missing security groups from a source which are not in a slice of security groups
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
func ConvertParamsToMap(params string) map[string]interface{} {
	if params == "" {
		return make(map[string]interface{})
	}
	var paramsTemplate interface{}
	json.Unmarshal([]byte(params), &paramsTemplate)
	return paramsTemplate.(map[string]interface{})
}
func ConvertMapToParams(data map[string]interface{}) string {
	if len(data) == 0 {
		return ""
	}
	b, _ := json.Marshal(data)
	return string(b)
}
