package resources

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"strings"
)

type CfIsolationSegmentSpaceResource struct{}

func (c CfIsolationSegmentSpaceResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	d.SetId(c.generateId(d))

	_, _, err := client.CCv3Client().UpdateSpaceIsolationSegmentRelationship(d.Get("space_id").(string), d.Get("segment_id").(string))
	return err
}

func (c CfIsolationSegmentSpaceResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	spaceId, _ := c.retrieveFromId(d.Id())
	r, _, err := client.CCv3Client().GetSpaceIsolationSegment(spaceId)
	if err != nil {
		return err
	}
	d.Set("segment_id", r.GUID)
	return nil
}

func (c CfIsolationSegmentSpaceResource) Update(*schema.ResourceData, interface{}) error {
	return nil
}

func (c CfIsolationSegmentSpaceResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	d.SetId(c.generateId(d))

	_, _, err := client.CCv3Client().UpdateSpaceIsolationSegmentRelationship(d.Get("space_id").(string), "")
	return err
}

func (c CfIsolationSegmentSpaceResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	if d.Id() == "" {
		return false, nil
	}
	client := meta.(cf_client.Client)
	spaceId, segmentId := c.retrieveFromId(d.Id())
	r, _, err := client.CCv3Client().GetSpaceIsolationSegment(spaceId)
	if err != nil {
		return false, err
	}
	return r.GUID == segmentId, nil
}

func (c CfIsolationSegmentSpaceResource) generateId(d *schema.ResourceData) string {
	return d.Get("space_id").(string) + "/" + d.Get("segment_id").(string)
}

func (c CfIsolationSegmentSpaceResource) retrieveFromId(id string) (spaceId, segmentId string) {
	infos := strings.Split(id, "/")
	spaceId = infos[0]
	if len(infos) > 1 {
		segmentId = infos[1]
	}
	return
}

func (c CfIsolationSegmentSpaceResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"segment_id": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"space_id": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
	}
}
