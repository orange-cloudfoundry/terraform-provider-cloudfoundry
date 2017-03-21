package resources

import "github.com/hashicorp/terraform/helper/schema"

type CfResource interface {
	Create(*schema.ResourceData, interface{}) error
	Read(*schema.ResourceData, interface{}) error
	Update(*schema.ResourceData, interface{}) error
	Delete(*schema.ResourceData, interface{}) error
	Exists(*schema.ResourceData, interface{}) (bool, error)
	Schema() map[string]*schema.Schema
}

func LoadCfResource(cfResource CfResource) *schema.Resource {
	return &schema.Resource{
		Create: cfResource.Create,
		Read:   cfResource.Read,
		Update: cfResource.Update,
		Delete: cfResource.Delete,
		Exists: cfResource.Exists,
		Schema: cfResource.Schema(),
	}
}
