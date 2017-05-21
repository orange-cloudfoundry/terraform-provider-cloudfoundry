package resources

import (
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"log"
)

type CfRouteResource struct {
	CfResource
}

func (c CfRouteResource) resourceObject(d *schema.ResourceData) models.Route {
	return models.Route{
		GUID: d.Id(),
		Host: d.Get("hostname").(string),
		Path: d.Get("path").(string),
		Port: d.Get("port").(int),
		Domain: models.DomainFields{
			GUID: d.Get("domain_id").(string),
		},
		Space: models.SpaceFields{
			GUID: d.Get("space_id").(string),
		},
	}
}
func (c CfRouteResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	route := c.resourceObject(d)
	var routeCf models.Route
	var err error
	if ok, _ := c.Exists(d, meta); ok {
		log.Printf(
			"[INFO] skipping creation of route %s/%s because it already exists on your Cloud Foundry",
			client.Config().ApiEndpoint,
			route.URL(),
		)
		return nil
	}

	port, randomPort := c.getPortOption(route)
	routeCf, err = client.Route().CreateInSpace(
		route.Host,
		route.Path,
		route.Domain.GUID,
		route.Space.GUID,
		port,
		randomPort,
	)
	if err != nil {
		return err
	}
	d.SetId(routeCf.GUID)

	return nil
}
func (c CfRouteResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	route := c.resourceObject(d)
	routeCf, err := client.Finder().GetRouteFromCf(d.Id())
	if err != nil {
		return err
	}
	if routeCf.GUID == "" {
		log.Printf(
			"[WARN] removing route %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			route.URL(),
		)
		d.SetId("")
		return nil
	}
	d.Set("hostname", routeCf.Host)
	d.Set("path", routeCf.Path)
	d.Set("domain_id", routeCf.Domain.GUID)
	d.Set("space_id", routeCf.Space.GUID)
	if routeCf.Port == 0 {
		d.Set("port", -1)
		return nil
	}
	if route.Port != 0 && routeCf.Port != route.Port {
		d.Set("port", routeCf.Port)
	}

	return nil

}
func (c CfRouteResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	if d.Id() != "" {
		d, err := client.Finder().GetRouteFromCf(d.Id())
		if err != nil {
			return false, err
		}
		return d.GUID != "", nil
	}
	route := c.resourceObject(d)
	port, _ := c.getPortOption(route)
	routeFinal, err := client.Route().Find(route.Host, route.Domain, route.Path, port)
	if err != nil {
		if _, ok := err.(*errors.ModelNotFoundError); ok {
			return false, nil
		}
		return false, err
	}
	if routeFinal.Space.GUID != route.Space.GUID {
		fmt.Errorf("Route '%s' has been already set on a different space.", route.URL())
	}
	d.SetId(routeFinal.GUID)
	return true, nil
}
func (c CfRouteResource) getPortOption(route models.Route) (port int, randomPort bool) {
	port = route.Port
	if port == 0 {
		randomPort = true
	}
	if port <= -1 {
		port = 0
	}
	return
}

//func (c CfRouteResource) updateBinding(client cf_client.Client, routeGuid, serviceGuid string) error {
//	client.RouteServiceBinding().Bind()
//}
func (c CfRouteResource) Update(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	route := c.resourceObject(d)
	routeCf, err := client.Finder().GetRouteFromCf(d.Id())
	if err != nil {
		return err
	}
	if routeCf.GUID == "" {
		log.Printf(
			"[WARN] removing route %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			route.URL(),
		)
		d.SetId("")
		return nil
	}
	if err := c.Delete(d, meta); err != nil {
		return err
	}
	if err := c.Create(d, meta); err != nil {
		return err
	}
	return nil
}
func (c CfRouteResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	return client.Route().Delete(d.Id())
}
func (c CfRouteResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"domain_id": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"space_id": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"hostname": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
		},
		"path": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
		},
		"port": &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
			Default:  -1,
			ForceNew: true,
		},
		"service_id": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"service_params": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
	}
}
