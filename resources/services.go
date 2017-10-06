package resources

import (
	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"github.com/viant/toolbox"
	"log"
)

type CfServiceResource struct{}

func (c CfServiceResource) resourceObject(d *schema.ResourceData) models.ServiceInstance {
	tagsSchema := d.Get("tags").(*schema.Set)
	tags := make([]string, 0)
	for _, tag := range tagsSchema.List() {
		tags = append(tags, tag.(string))
	}
	return models.ServiceInstance{
		ServiceInstanceFields: models.ServiceInstanceFields{
			GUID:            d.Id(),
			Name:            d.Get("name").(string),
			Tags:            tags,
			Params:          ConvertParamsToMap(d.Get("params").(string)),
			SysLogDrainURL:  d.Get("syslog_drain_url").(string),
			RouteServiceURL: d.Get("route_service_url").(string),
		},
	}
}

func (c CfServiceResource) findPlanGuid(client cf_client.Client, service, plan string) (planGuid string, err error) {
	planGuid, err = client.Services().FindServicePlanByDescription(resources.ServicePlanDescription{
		ServiceLabel:    service,
		ServicePlanName: plan,
	})
	return
}
func (c CfServiceResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	svc := c.resourceObject(d)
	isUserProvided := d.Get("user_provided").(bool)
	client.Gateways().Config.SetSpaceFields(models.SpaceFields{
		GUID: d.Get("space_id").(string),
	})
	var err error
	var planGuid string
	if ok, _ := c.Exists(d, meta); ok {
		log.Printf(
			"[INFO] skipping creation of service %s/%s because it already exists on your Cloud Foundry",
			client.Config().ApiEndpoint,
			svc.Name,
		)
	} else {
		if !isUserProvided {
			planGuid, err = c.findPlanGuid(client, d.Get("service").(string), d.Get("plan").(string))
			if err != nil {
				return err
			}
			err = client.Services().CreateServiceInstance(svc.Name, planGuid, svc.Params, svc.Tags)
		} else {
			err = client.UserProvidedService().Create(
				svc.Name,
				svc.SysLogDrainURL,
				svc.RouteServiceURL,
				svc.Params,
			)
		}
		if err != nil {
			return err
		}
		c.Exists(d, meta)
	}
	svcCf, err := client.Finder().GetServiceFromCf(d.Id())
	if err != nil {
		return err
	}
	svc.GUID = d.Id()
	d.Set("plan_id", svcCf.ServicePlan.GUID)
	if !isUserProvided && (svcCf.ServicePlan.GUID != planGuid || c.isTagsDiff(svcCf.Tags, svc.Tags)) {
		err = client.Services().UpdateServiceInstance(
			d.Id(),
			planGuid,
			ConvertParamsToMap(d.Get("update_params").(string)),
			svc.Tags,
		)
	}
	if isUserProvided &&
		(svcCf.RouteServiceURL != svc.RouteServiceURL || svcCf.SysLogDrainURL != svc.SysLogDrainURL) {
		err = client.UserProvidedService().Update(
			svc.ServiceInstanceFields,
		)
	}
	return err
}
func (c CfServiceResource) isTagsDiff(currentTags, wantedTags []string) bool {
	if len(currentTags) != len(wantedTags) {
		return true
	}
	for _, tag := range currentTags {
		if !toolbox.HasSliceAnyElements(wantedTags, tag) {
			return true
		}
	}
	for _, tag := range wantedTags {
		if !toolbox.HasSliceAnyElements(currentTags, tag) {
			return true
		}
	}
	return false
}
func (c CfServiceResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	client.Gateways().Config.SetSpaceFields(models.SpaceFields{
		GUID: d.Get("space_id").(string),
	})
	svc := c.resourceObject(d)
	svcCf, err := client.Finder().GetServiceFromCf(d.Id())
	if err != nil {
		return err
	}
	if svcCf.GUID == "" {
		log.Printf(
			"[WARN] removing service %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			svc.Name,
		)
		d.SetId("")
		return nil
	}
	d.Set("name", svcCf.Name)
	tagsSchema := schema.NewSet(d.Get("tags").(*schema.Set).F, make([]interface{}, 0))
	for _, tag := range svcCf.Tags {
		tagsSchema.Add(tag)
	}
	d.Set("tags", tagsSchema)
	d.Set("user_provided", svcCf.IsUserProvided())
	d.Set("route_service_url", svcCf.RouteServiceURL)
	d.Set("syslog_drain_url", svcCf.SysLogDrainURL)
	d.Set("plan_id", svcCf.ServicePlan.GUID)

	return nil

}
func (c CfServiceResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	if d.Id() != "" {
		d, err := client.Finder().GetServiceFromCf(d.Id())
		if err != nil {
			return false, err
		}
		return d.GUID != "", nil
	}
	client.Gateways().Config.SetSpaceFields(models.SpaceFields{
		GUID: d.Get("space_id").(string),
	})
	instance := c.resourceObject(d)
	instanceCf, err := client.Services().FindInstanceByName(instance.Name)
	if err != nil {
		if _, ok := err.(*errors.ModelNotFoundError); ok {
			return false, nil
		}
		return false, err
	}
	d.SetId(instanceCf.GUID)
	return true, nil
}

func (c CfServiceResource) Update(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	client.Gateways().Config.SetSpaceFields(models.SpaceFields{
		GUID: d.Get("space_id").(string),
	})
	svc := c.resourceObject(d)
	svcCf, err := client.Finder().GetServiceFromCf(d.Id())
	if err != nil {
		return err
	}
	if svcCf.GUID == "" {
		log.Printf(
			"[WARN] removing service %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			svc.Name,
		)
		d.SetId("")
		return nil
	}
	d.Set("plan_id", svcCf.ServicePlan.GUID)
	if svc.IsUserProvided() {
		return client.UserProvidedService().Update(svc.ServiceInstanceFields)
	}
	planGuid, err := c.findPlanGuid(client, d.Get("service").(string), d.Get("plan").(string))
	if err != nil {
		return err
	}
	return client.Services().UpdateServiceInstance(
		d.Id(),
		planGuid,
		ConvertParamsToMap(d.Get("update_params").(string)),
		svc.Tags,
	)
}
func (c CfServiceResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	svc := c.resourceObject(d)
	bindings, err := client.ServiceBinding().ListAllForService(d.Id())
	if err != nil {
		return err
	}
	svc.ServiceBindings = bindings
	for _, binding := range bindings {
		_, err = client.ServiceBinding().Delete(svc, binding.AppGUID)
		if err != nil {
			return err
		}
	}
	svc.ServiceBindings = make([]models.ServiceBindingFields, 0)
	return client.Services().DeleteService(svc)
}
func (c CfServiceResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"space_id": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"service": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"plan": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"plan_id": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
		"params": &schema.Schema{
			Type:      schema.TypeString,
			Optional:  true,
			Sensitive: true,
		},
		"update_params": &schema.Schema{
			Type:      schema.TypeString,
			Optional:  true,
			Sensitive: true,
		},
		"user_provided": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
			ForceNew: true,
		},
		"route_service_url": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"syslog_drain_url": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"tags": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Set:      schema.HashString,
		},
	}
}
func (c CfServiceResource) DataSourceSchema() map[string]*schema.Schema {
	return CreateDataSourceSchema(c, "name", "space_id")
}
func (c CfServiceResource) DataSourceRead(d *schema.ResourceData, meta interface{}) error {
	fn := CreateDataSourceReadFunc(c)
	return fn(d, meta)
}
