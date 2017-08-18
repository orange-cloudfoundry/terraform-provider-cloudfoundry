package resources

import (
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"github.com/viant/toolbox"
	"log"
)

type CfSpaceResource struct {
	CfResource
}

func (c CfSpaceResource) resourceObject(d *schema.ResourceData) models.Space {
	spaceField := models.SpaceFields{
		GUID:     d.Id(),
		Name:     d.Get("name").(string),
		AllowSSH: d.Get("allow_ssh").(bool),
	}
	orgField := models.OrganizationFields{
		GUID: d.Get("org_id").(string),
	}
	repo := models.Space{
		SpaceFields:    spaceField,
		Organization:   orgField,
		SpaceQuotaGUID: d.Get("quota_id").(string),
		SecurityGroups: c.extractSecGroups(d),
	}

	return repo
}
func extractSecGroupsFromSet(secGroupsSet *schema.Set) []models.SecurityGroupFields {
	secGroups := make([]models.SecurityGroupFields, 0)
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
func (c CfSpaceResource) extractSecGroups(d *schema.ResourceData) []models.SecurityGroupFields {
	return extractSecGroupsFromSet(d.Get("sec_groups").(*schema.Set))
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
		spaceCf, err = client.Finder().GetSpaceFromCf(d.Id())
	} else {
		spaceCf, err = client.Spaces().Create(space.Name, space.Organization.GUID, space.SpaceQuotaGUID)
	}
	if err != nil {
		return err
	}
	spaceCf.SecurityGroups = c.filterSecGroup(client, spaceCf.SecurityGroups, space.SecurityGroups)
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
	missingSecGroupsInTo := GetMissingSecGroup(secGroupFrom, secGroupTo)
	if len(missingSecGroupsInFrom) == 0 && len(missingSecGroupsInTo) == 0 {
		return nil
	}
	for _, secGroup := range missingSecGroupsInFrom {
		err := client.SecurityGroupsSpaceBinder().BindSpace(secGroup.GUID, spaceId)
		if err != nil {
			return err
		}
	}
	for _, secGroup := range missingSecGroupsInTo {
		err := client.SecurityGroupsSpaceBinder().UnbindSpace(secGroup.GUID, spaceId)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c CfSpaceResource) filterSecGroup(client cf_client.Client, secGroupFields, secGroupFieldsFromTf []models.SecurityGroupFields) []models.SecurityGroupFields {
	secGroupsFiltered := make([]models.SecurityGroupFields, 0)
	for _, secGroupField := range secGroupFields {
		secGroup, _ := client.Finder().GetSecGroupFromCf(secGroupField.GUID)
		if secGroup.GUID == "" || len(secGroup.Spaces) == 0 {
			continue
		}
		secGroupsFiltered = append(secGroupsFiltered, secGroupField)
	}
	return c.filterSecGroupByTerraformExist(client, secGroupsFiltered, secGroupFieldsFromTf)
}
func (c CfSpaceResource) filterSecGroupByTerraformExist(client cf_client.Client, secGroupFields, secGroupFieldsFromTf []models.SecurityGroupFields) []models.SecurityGroupFields {
	finalSecGroups := make([]models.SecurityGroupFields, 0)
	toolbox.FilterSliceElements(secGroupFields, func(item models.SecurityGroupFields) bool {
		for _, secGroup := range secGroupFieldsFromTf {
			if item.GUID == secGroup.GUID {
				return true
			}
		}
		return false
	}, &finalSecGroups)
	return finalSecGroups
}
func (c CfSpaceResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	name := d.Get("name").(string)
	space, err := client.Finder().GetSpaceFromCf(d.Id())
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
	currentSecGroups := extractSecGroupsFromSet(d.Get("sec_groups").(*schema.Set))
	space.SecurityGroups = c.filterSecGroup(client, space.SecurityGroups, currentSecGroups)
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
	spaceCf, err := client.Finder().GetSpaceFromCf(space.GUID)
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
	if d.HasChange("sec_groups") {
		currentTfSecGroups, _ := d.GetChange("sec_groups")
		spaceCf.SecurityGroups = c.filterSecGroupByTerraformExist(client, spaceCf.SecurityGroups, extractSecGroupsFromSet(currentTfSecGroups.(*schema.Set)))
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
	if d.Id() != "" {
		d, err := client.Finder().GetSpaceFromCf(d.Id())
		if err != nil {
			return false, err
		}
		return d.GUID != "", nil
	}
	space, err := client.Spaces().FindByNameInOrg(d.Get("name").(string), d.Get("org_id").(string))
	if err != nil {
		if _, ok := err.(*errors.ModelNotFoundError); ok {
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
			Default:  true,
		},
	}
}
func (c CfSpaceResource) DataSourceSchema() map[string]*schema.Schema {
	return CreateDataSourceSchema(c, "name", "org_id")
}
func (c CfSpaceResource) DataSourceRead(d *schema.ResourceData, meta interface{}) error {
	fn := CreateDataSourceReadFunc(c)
	return fn(d, meta)
}
