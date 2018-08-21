package resources

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"strings"
)

type CfIsolationSegmentsEntitlementResource struct{}

func (c CfIsolationSegmentsEntitlementResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"segment_id": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"org_id": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"default": &schema.Schema{
			Type:     schema.TypeBool,
			Default:  false,
			Optional: true,
		},
	}
}

func (c CfIsolationSegmentsEntitlementResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	d.SetId(c.generateId(d))

	_, _, err := client.CCv3Client().EntitleIsolationSegmentToOrganizations(d.Get("segment_id").(string), []string{d.Get("org_id").(string)})
	if err != nil {
		return err
	}
	return c.updateDefaultIsolationSegment(d, meta)
}

func (c CfIsolationSegmentsEntitlementResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	orgId, segmentId := c.retrieveFromId(d.Id())
	orgs, _, err := client.CCv3Client().GetIsolationSegmentOrganizations(segmentId)
	if err != nil {
		return err
	}
	orgExists := false
	for _, org := range orgs {
		if org.GUID == orgId {
			orgExists = true
			break
		}
	}
	if !orgExists {
		d.Set("org_id", "")
		return nil
	}
	r, _, err := client.CCv3Client().GetOrganizationDefaultIsolationSegment(orgId)
	if err != nil {
		return err
	}
	if r.GUID == segmentId {
		d.Set("default", true)
	} else {
		d.Set("default", false)
	}
	return nil
}

func (c CfIsolationSegmentsEntitlementResource) generateId(d *schema.ResourceData) string {
	return d.Get("org_id").(string) + "/" + d.Get("segment_id").(string)
}

func (c CfIsolationSegmentsEntitlementResource) retrieveFromId(id string) (orgId, segmentId string) {
	infos := strings.Split(id, "/")
	orgId = infos[0]
	if len(infos) > 1 {
		segmentId = infos[1]
	}
	return
}

func (c CfIsolationSegmentsEntitlementResource) Update(d *schema.ResourceData, meta interface{}) error {
	if !d.HasChange("default") {
		return nil
	}
	return c.updateDefaultIsolationSegment(d, meta)
}

func (c CfIsolationSegmentsEntitlementResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	if d.Id() == "" {
		return false, nil
	}
	client := meta.(cf_client.Client)
	orgId, segmentId := c.retrieveFromId(d.Id())
	orgs, _, err := client.CCv3Client().GetIsolationSegmentOrganizations(segmentId)
	if err != nil {
		return false, err
	}
	for _, org := range orgs {
		if org.GUID == orgId {
			return true, nil
		}
	}
	return false, nil
}

func (c CfIsolationSegmentsEntitlementResource) updateDefaultIsolationSegment(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	if d.Get("default").(bool) {
		_, _, err := client.CCv3Client().UpdateOrganizationDefaultIsolationSegmentRelationship(d.Get("org_id").(string), d.Get("segment_id").(string))
		return err
	}
	_, _, err := client.CCv3Client().UpdateOrganizationDefaultIsolationSegmentRelationship(d.Get("org_id").(string), "")
	return err
}

func (c CfIsolationSegmentsEntitlementResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	client.CCv3Client().UpdateOrganizationDefaultIsolationSegmentRelationship(d.Get("org_id").(string), "")
	_, err := client.CCv3Client().DeleteIsolationSegmentOrganization(d.Get("segment_id").(string), d.Get("org_id").(string))
	return err
}
