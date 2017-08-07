package resources

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/bitsmanager"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
)

type CfAppsResource struct {
	CfResource
}

func (c CfAppsResource) MakeBitsManager(meta interface{}) bitsmanager.BitsManager {
	client := meta.(cf_client.Client)
	localHandler := bitsmanager.LocalHandler{}
	httpHandler := bitsmanager.HttpHandler{
		SkipInsecureSSL: client.Config().SkipInsecureSSL,
	}
	return bitsmanager.NewCloudControllerBitsManager(
		client.ApplicationBits(),
		[]bitsmanager.Handler{localHandler, httpHandler},
	)
}
func (c CfAppsResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	bm := c.MakeBitsManager(meta)
	err := bm.Upload(d.Get("app_guid").(string), d.Get("path").(string))
	if err != nil {
		return err
	}
	localSha1, err := bm.GetSha1(d.Get("path").(string))
	if err != nil {
		return err
	}
	d.Set("path_sha1", localSha1)
	rmtSha1, err := client.ApplicationBits().GetApplicationSha1(d.Get("app_guid").(string))
	if err != nil {
		return err
	}
	d.Set("remote_sha1", rmtSha1)
	d.SetId("myid")
	return nil
}
func (c CfAppsResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	bm := c.MakeBitsManager(meta)
	isDiffLocal, sha1Local, err := bm.IsDiff(d.Get("path").(string), d.Get("path_sha1").(string))
	if err != nil {
		return err
	}
	isDiffRmt, sha1Rmt, err := client.ApplicationBits().IsDiff(d.Get("app_guid").(string), d.Get("remote_sha1").(string))
	if err != nil {
		return err
	}
	if isDiffLocal || isDiffRmt {
		d.Set("path_sha1", sha1Local)
		d.Set("remote_sha1", sha1Rmt)
		d.Set("bits_has_changed", "modified")
	} else {
		d.Set("bits_has_changed", "")
	}

	return nil
}
func (c CfAppsResource) Update(d *schema.ResourceData, meta interface{}) error {
	d.Set("bits_has_changed", "")
	return c.Create(d, meta)
}
func (c CfAppsResource) Delete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
func (c CfAppsResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	if d.Id() != "" {
		return true, nil
	}
	return false, nil
}
func (c CfAppsResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"app_guid": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"path": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"path_sha1": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
		"remote_sha1": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
		"bits_has_changed": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
	}
}
