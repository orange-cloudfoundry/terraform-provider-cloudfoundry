package resources

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/common"
	"github.com/viant/toolbox"
	"log"
)

type CfIsolationSegmentsResource struct{}
type IsolationSegment struct {
	ccv3.IsolationSegment
	OrgsGUID []string
}

func (c CfIsolationSegmentsResource) resourceObject(d *schema.ResourceData) *IsolationSegment {
	segment := &IsolationSegment{
		IsolationSegment: ccv3.IsolationSegment{
			Name: d.Get("name").(string),
			GUID: d.Id(),
		},
	}
	orgsSchema := d.Get("orgs_id").(*schema.Set)
	orgs := make([]string, 0)
	for _, org := range orgsSchema.List() {
		orgs = append(orgs, org.(string))
	}
	segment.OrgsGUID = orgs
	return segment
}
func (c CfIsolationSegmentsResource) updateIsolationSegmentToOrg(client cf_client.Client, isoGuid string, currentOrgsId, wantedOrgsId []string) error {
	toCreate := make([]string, 0)
	toDelete := make([]string, 0)
	for _, orgId := range wantedOrgsId {
		if !toolbox.HasSliceAnyElements(currentOrgsId, orgId) {
			toCreate = append(toCreate, orgId)
		}
	}
	for _, orgId := range currentOrgsId {
		if !toolbox.HasSliceAnyElements(wantedOrgsId, orgId) {
			toDelete = append(toDelete, orgId)
		}
	}
	for _, orgId := range toDelete {
		_, err := client.CCv3Client().DeleteIsolationSegmentOrganization(isoGuid, orgId)
		if err != nil {
			return err
		}
	}
	if len(toCreate) == 0 {
		return nil
	}
	_, _, err := client.CCv3Client().EntitleIsolationSegmentToOrganizations(isoGuid, toCreate)
	if err != nil {
		return err
	}
	return nil
}
func (c CfIsolationSegmentsResource) retrieveOrgsIdFromIsolationSegment(client cf_client.Client, isoGuid string) ([]string, error) {
	orgsId := make([]string, 0)
	orgs, _, err := client.CCv3Client().GetIsolationSegmentOrganizations(isoGuid)
	if err != nil {
		return orgsId, err
	}
	for _, org := range orgs {
		orgsId = append(orgsId, org.GUID)
	}
	return orgsId, nil
}
func (c CfIsolationSegmentsResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	segment := c.resourceObject(d)
	if ok, _ := c.Exists(d, meta); ok {
		log.Printf(
			"[INFO] skipping creation of isolation segment %s/%s because it already exists on your Cloud Foundry",
			client.Config().ApiEndpoint,
			segment.Name,
		)
	} else {
		segment, _, err := client.CCv3Client().CreateIsolationSegment(segment.IsolationSegment)
		if err != nil {
			return err
		}
		d.SetId(segment.GUID)
	}

	return c.updateIsolationSegmentToOrg(client, d.Id(), make([]string, 0), segment.OrgsGUID)
}

func (c CfIsolationSegmentsResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	segmentCf, _, err := client.CCv3Client().GetIsolationSegment(d.Id())
	segment := c.resourceObject(d)
	if err != nil {
		if _, ok := err.(ccerror.ResourceNotFoundError); ok {
			log.Printf(
				"[WARN] removing isolation segment %s/%s from state because it no longer exists in your Cloud Foundry",
				client.Config().ApiEndpoint,
				segment.Name,
			)
			d.SetId("")
			return nil
		}
		return err
	}
	d.Set("name", segmentCf.Name)

	return nil
}
func (c CfIsolationSegmentsResource) Update(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	segment := c.resourceObject(d)
	_, _, err := client.CCv3Client().GetIsolationSegment(d.Id())
	if err != nil {
		if _, ok := err.(ccerror.ResourceNotFoundError); ok {
			log.Printf(
				"[WARN] removing isolation segment %s/%s from state because it no longer exists in your Cloud Foundry",
				client.Config().ApiEndpoint,
				segment.Name,
			)
			d.SetId("")
			return nil
		}
		return err
	}
	if !d.HasChange("orgs_id") {
		return nil
	}
	current, _ := d.GetChange("orgs_id")

	return c.updateIsolationSegmentToOrg(client, d.Id(), common.SchemaSetToStringList(current.(*schema.Set)), segment.OrgsGUID)
}

func (c CfIsolationSegmentsResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	_, err := client.CCv3Client().DeleteIsolationSegment(d.Id())
	return err
}

func (c CfIsolationSegmentsResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	if d.Id() != "" {
		d, _, err := client.CCv3Client().GetIsolationSegment(d.Id())

		if err != nil {
			if _, ok := err.(ccerror.ResourceNotFoundError); ok {
				return false, nil
			}
			return false, err
		}
		return d.GUID != "", nil
	}
	name := d.Get("name").(string)
	segment, _, err := client.CCv3Client().GetIsolationSegments(ccv3.Query{
		Key:    "q",
		Values: []string{"name:" + name},
	})
	if err != nil {
		if _, ok := err.(ccerror.ResourceNotFoundError); ok {
			return false, nil
		}
		return false, err
	}
	if len(segment) == 0 {
		return false, err
	}
	d.SetId(segment[0].GUID)
	return true, nil
}

func (c CfIsolationSegmentsResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"orgs_id": &schema.Schema{
			Type:       schema.TypeSet,
			Optional:   true,
			Deprecated: "use entitle_isolation_segment instead",
			Elem:       &schema.Schema{Type: schema.TypeString},
			Set:        schema.HashString,
		},
	}
}
func (c CfIsolationSegmentsResource) DataSourceSchema() map[string]*schema.Schema {
	return CreateDataSourceSchema(c, "name")
}
func (c CfIsolationSegmentsResource) DataSourceRead(d *schema.ResourceData, meta interface{}) error {
	fn := CreateDataSourceReadFuncWithReq(c, "name")
	return fn(d, meta)
}
