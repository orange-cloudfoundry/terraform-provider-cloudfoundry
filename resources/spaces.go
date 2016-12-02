package resources

import (
	"github.com/hashicorp/terraform/helper/schema"
	"code.cloudfoundry.org/cli/cf/models"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"strings"
	"log"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/resources/caching"
)

type CfSpaceResource struct {
	CfResource
}

func NewCfSpaceResource() CfResource {
	return &CfSpaceResource{}
}
func (c CfSpaceResource) resourceObject(d *schema.ResourceData) models.Space {
	spaceField := models.SpaceFields{
		GUID: d.Id(),
		Name: d.Get("name").(string),
		AllowSSH: d.Get("allow_ssh").(bool),
	}
	orgField := models.OrganizationFields{
		GUID: d.Get("org_id").(string),
	}
	repo := models.Space{
		SpaceFields: spaceField,
		Organization: orgField,
		SpaceQuotaGUID: d.Get("quota_id").(string),
		SecurityGroups: c.extractSecGroups(d),
	}

	return repo
}
func (c CfSpaceResource) extractSecGroups(d *schema.ResourceData) []models.SecurityGroupFields {
	secGroups := make([]models.SecurityGroupFields, 0)
	secGroupsSet := d.Get("sec_groups").(*schema.Set)
	for _, secGroup := range secGroupsSet.List() {
		secGroups = append(
			secGroups,
			models.SecurityGroupFields{
				GUID: secGroup.(string),
			},
		)
	}
	return secGroups
}
func (c CfSpaceResource) getSpaceFromCf(client cf_client.Client, orgGuid, spaceGuid string) (models.Space, error) {
	var space models.Space
	err := client.Spaces().ListSpacesFromOrg(orgGuid, func(spaceCf models.Space) bool {
		if spaceCf.GUID == spaceGuid {
			space = spaceCf
			return false
		}
		return true
	})
	if err != nil && strings.Contains(err.Error(), "status code: 404") {
		return models.Space{}, nil
	}
	return space, err
}
func (c CfSpaceResource) Create(d *schema.ResourceData, meta interface{}) error {
	var spaceCf models.Space
	var err error
	client := meta.(cf_client.Client)
	space := c.resourceObject(d)

	if ok, _ := c.Exists(d, meta); ok {
		log.Printf(
			"[INFO] skipping creation of space %s/%s because it already exists on your Cloud Foundry",
			client.Config().ApiEndpoint,
			space.Name,
		)
		spaceCf, err = c.getSpaceFromCf(client, space.Organization.GUID, d.Id())
	} else {
		spaceCf, err = client.Spaces().Create(space.Name, space.Organization.GUID, space.SpaceQuotaGUID)
	}
	if err != nil {
		return err
	}
	spaceCf.SecurityGroups = c.filterSecGroup(client, spaceCf.SecurityGroups)
	err = c.updateSecGroups(client, spaceCf.SecurityGroups, space.SecurityGroups, spaceCf.GUID)
	if err != nil {
		return err
	}
	d.Set("quota_id", spaceCf.SpaceQuotaGUID)
	d.SetId(spaceCf.GUID)
	err = client.Spaces().SetAllowSSH(spaceCf.GUID, space.AllowSSH)
	if err != nil {
		return err
	}
	return nil
}
func (c CfSpaceResource) updateSecGroups(client cf_client.Client, secGroupFrom, secGroupTo []models.SecurityGroupFields, spaceId string) error {
	missingSecGroupsInFrom := GetMissingSecGroup(secGroupTo, secGroupFrom)
	if len(missingSecGroupsInFrom) == 0 {
		return nil
	}
	for _, secGroup := range missingSecGroupsInFrom {
		err := client.SecurityGroupsSpaceBinder().BindSpace(secGroup.GUID, spaceId)
		if err != nil {
			return err
		}
	}
	missingSecGroupsInTo := GetMissingSecGroup(secGroupFrom, secGroupTo)
	for _, secGroup := range missingSecGroupsInTo {
		err := client.SecurityGroupsSpaceBinder().UnbindSpace(secGroup.GUID, spaceId)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c CfSpaceResource) filterSecGroup(client cf_client.Client, secGroupFields []models.SecurityGroupFields) []models.SecurityGroupFields {
	secGroupsFiltered := make([]models.SecurityGroupFields, 0)
	for _, secGroupField := range secGroupFields {
		secGroup, _ := caching.GetSecGroupFromCf(client, secGroupField.GUID, false)
		if secGroup.GUID == "" || len(secGroup.Spaces) == 0 {
			continue
		}
		secGroupsFiltered = append(secGroupsFiltered, secGroupField)
	}
	return secGroupsFiltered
}
func (c CfSpaceResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	name := d.Get("name").(string)
	orgGuid := d.Get("org_id").(string)
	space, err := c.getSpaceFromCf(client, orgGuid, d.Id())
	if err != nil {
		return err
	}
	if space.GUID == "" {
		log.Printf(
			"[WARN] removing space %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			name,
		)
		d.SetId("")
		return nil
	}
	space.SecurityGroups = c.filterSecGroup(client, space.SecurityGroups)
	d.Set("quota_id", space.SpaceQuotaGUID)
	secGroupsSchema := schema.NewSet(d.Get("sec_groups").(*schema.Set).F, make([]interface{}, 0))
	for _, secGroup := range space.SecurityGroups {
		secGroupsSchema.Add(secGroup.GUID)
	}
	d.Set("sec_groups", secGroupsSchema)
	d.Set("allow_ssh", space.AllowSSH)
	return nil
}
func (c CfSpaceResource) Update(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	space := c.resourceObject(d)
	spaceCf, err := c.getSpaceFromCf(client, space.Organization.GUID, space.GUID)
	if err != nil {
		return err
	}
	if space.GUID == "" {
		log.Printf(
			"[WARN] removing space %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			space.Name,
		)
		d.SetId("")
		return nil
	}
	if spaceCf.AllowSSH != space.AllowSSH {
		client.Spaces().SetAllowSSH(space.GUID, space.AllowSSH)
	}
	err = c.updateSecGroups(client, spaceCf.SecurityGroups, space.SecurityGroups, spaceCf.GUID)
	if err != nil {
		return err
	}
	err = c.bindOrUnbindQuota(client, space.GUID, spaceCf.SpaceQuotaGUID, space.SpaceQuotaGUID)
	if err != nil {
		return err
	}
	return nil
}
func (c CfSpaceResource) bindOrUnbindQuota(client cf_client.Client, spaceGuid, spaceQuotaFrom, spaceQuotaTo string) error {
	if spaceQuotaFrom == spaceQuotaTo {
		return nil
	}
	if spaceQuotaFrom == "" {
		return client.SpaceQuotas().AssociateSpaceWithQuota(spaceGuid, spaceQuotaTo)
	}
	if spaceQuotaTo == "" {
		return client.SpaceQuotas().UnassignQuotaFromSpace(spaceGuid, spaceQuotaFrom)
	}
	err := client.SpaceQuotas().UnassignQuotaFromSpace(spaceGuid, spaceQuotaFrom)
	if err != nil {
		return err
	}
	return client.SpaceQuotas().AssociateSpaceWithQuota(spaceGuid, spaceQuotaTo)
}
func (c CfSpaceResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	return client.Spaces().Delete(d.Id())
}
func (c CfSpaceResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	space, err := client.Spaces().FindByNameInOrg(d.Get("name").(string), d.Get("org_id").(string))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	d.SetId(space.GUID)
	d.Set("quota_id", space.SpaceQuotaGUID)
	return true, nil
}
func (c CfSpaceResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"quota_id": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"sec_groups": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Set:      schema.HashString,
		},
		"org_id": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"allow_ssh": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
			Default: true,
		},
	}
}

