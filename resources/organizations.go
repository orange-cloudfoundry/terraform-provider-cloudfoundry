package resources

import (
	"code.cloudfoundry.org/cli/cf/models"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"log"
	"strings"
)

const DEFAULT_ORG_QUOTA_NAME = "default"

type CfOrganizationResource struct {
	CfResource
}

func NewCfOrganizationResource() CfResource {
	return &CfOrganizationResource{}
}
func (c CfOrganizationResource) resourceObject(d *schema.ResourceData) models.Organization {
	quotaDef := models.QuotaFields{
		GUID: d.Get("quota_id").(string),
	}
	orgField := models.OrganizationFields{
		GUID:            d.Id(),
		Name:            d.Get("name").(string),
		QuotaDefinition: quotaDef,
	}
	repo := models.Organization{
		OrganizationFields: orgField,
	}

	return repo
}
func (c CfOrganizationResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	org := c.resourceObject(d)
	if ok, _ := c.Exists(d, meta); ok {
		log.Printf(
			"[INFO] skipping creation of organization %s/%s because it already exists on your Cloud Foundry",
			client.Config().ApiEndpoint,
			org.Name,
		)
	} else {
		err := client.Organizations().Create(org)
		if err != nil {
			return err
		}
		c.Exists(d, meta)
	}
	orgCf, err := c.getOrgFromCf(client, d.Id())
	if err != nil {
		return err
	}
	err = c.bindQuota(client, org.QuotaDefinition.GUID, orgCf.GUID)
	if err != nil {
		return err
	}
	d.SetId(orgCf.GUID)
	return nil
}
func (c CfOrganizationResource) bindQuota(client cf_client.Client, quotaId, orgId string) error {
	if quotaId == "" {
		return nil
	}
	return client.Quotas().AssignQuotaToOrg(orgId, quotaId)
}
func (c CfOrganizationResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	orgName := d.Get("name").(string)
	org, err := c.getOrgFromCf(client, d.Id())
	if err != nil {
		return err
	}
	if org.GUID == "" {
		log.Printf(
			"[WARN] removing organization %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			orgName,
		)
		d.SetId("")
		return nil
	}
	quota, _ := client.Quotas().FindByName(DEFAULT_ORG_QUOTA_NAME)
	if quota.GUID == "" || org.QuotaDefinition.GUID != quota.GUID {
		d.Set("quota_id", org.QuotaDefinition.GUID)
	}
	d.Set("name", org.Name)
	return nil

}
func (c CfOrganizationResource) getOrgFromCf(client cf_client.Client, orgGuid string) (models.Organization, error) {
	orgs, err := client.Organizations().GetManyOrgsByGUID([]string{orgGuid})
	if err != nil {
		if strings.Contains(err.Error(), "status code: 404") {
			return models.Organization{}, nil
		}
		return models.Organization{}, err
	}
	return client.Organizations().FindByName(orgs[0].Name)
}
func (c CfOrganizationResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	orgName := d.Get("name").(string)
	org, err := client.Organizations().FindByName(orgName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	d.SetId(org.GUID)
	return true, nil
}
func (c CfOrganizationResource) Update(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	quotaId := d.Get("quota_id").(string)
	orgName := d.Get("name").(string)
	org, err := c.getOrgFromCf(client, d.Id())
	if err != nil {
		return err
	}
	if org.GUID == "" {
		log.Printf(
			"[WARN] removing organization %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			orgName,
		)
		d.SetId("")
		return nil
	}
	if org.QuotaDefinition.GUID != quotaId {
		err = c.bindQuota(client, quotaId, d.Id())
		if err != nil {
			return err
		}
	}
	if org.Name != orgName {
		err = client.Organizations().Rename(d.Id(), d.Get("name").(string))
		if err != nil {
			return err
		}
	}

	return nil
}
func (c CfOrganizationResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	isSystemDomain := d.Get("is_system_domain").(bool)
	orgName := d.Get("name").(string)
	if isSystemDomain {
		log.Printf(
			"[WARN] removing organization %s/%s isn't possible because it's the system_domain organization of your Cloud Foundry",
			client.Config().ApiEndpoint,
			orgName,
		)
		return nil
	}
	return client.Organizations().Delete(d.Id())
}
func (c CfOrganizationResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"quota_id": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"is_system_domain": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
	}
}
