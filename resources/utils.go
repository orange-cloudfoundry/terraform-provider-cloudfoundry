package resources

import (
	"code.cloudfoundry.org/cli/cf/models"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/viant/toolbox"
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
func CreateDataSourceSchema(resource CfResource, keysUntouch ...string) map[string]*schema.Schema {
	schemas := resource.Schema()
	if len(keysUntouch) == 0 {
		keysUntouch = append(keysUntouch, "name")
	}

	for key, resSchema := range schemas {
		resSchema.ForceNew = false
		if toolbox.HasSliceAnyElements(keysUntouch, key) {
			continue
		}
		resSchema.Default = nil
		resSchema.ValidateFunc = nil
		resSchema.Computed = true
		resSchema.Required = false
		resSchema.Optional = false
	}
	return schemas
}
func CreateDataSourceReadFunc(resource CfResource) func(d *schema.ResourceData, meta interface{}) error {
	return func(d *schema.ResourceData, meta interface{}) error {
		exists, err := resource.Exists(d, meta)
		if err != nil {
			return err
		}
		if !exists {
			name, hasName := d.GetOk("name")
			if !hasName {
				return fmt.Errorf("Can't found data source requested.")
			}
			return fmt.Errorf("Can't found data source requested with name '%s'.", name)
		}
		return resource.Read(d, meta)
	}
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
