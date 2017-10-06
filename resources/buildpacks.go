package resources

import (
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/common"
	"log"
	"os"
	"path"
	"path/filepath"
)

type CfBuildpackResource struct{}

func (c CfBuildpackResource) resourceObject(d *schema.ResourceData) (models.Buildpack, error) {
	var err error
	position := d.Get("position").(int)
	enabled := d.Get("enabled").(bool)
	locked := d.Get("locked").(bool)
	filename := d.Get("path").(string)
	if filename != "" {
		filename, err = c.generateFilename(d.Get("path").(string))
		if err != nil {
			return models.Buildpack{}, err
		}
	}

	return models.Buildpack{
		GUID:     d.Id(),
		Name:     d.Get("name").(string),
		Enabled:  &enabled,
		Locked:   &locked,
		Position: &position,
		Filename: filename,
	}, nil
}
func (c CfBuildpackResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	buildpack, err := c.resourceObject(d)
	if err != nil {
		return err
	}
	var buildpackCf models.Buildpack
	if ok, _ := c.Exists(d, meta); ok {
		log.Printf(
			"[INFO] skipping creation of buildpack %s/%s because it already exists on your Cloud Foundry",
			client.Config().ApiEndpoint,
			buildpack.Name,
		)
		buildpackCf, err = client.Finder().GetBuildpackFromCf(d.Id())
		if err != nil {
			return err
		}
	} else {
		buildpackCf, err = client.Buildpack().Create(buildpack.Name, buildpack.Position, buildpack.Enabled, buildpack.Locked)
		if err != nil {
			return err
		}
		c.Exists(d, meta)
	}
	buildpack.GUID = d.Id()
	if c.isSystemBuildpackManaged(buildpack) {
		return nil
	}
	return c.updateBuildpack(client, buildpackCf, buildpack, d.Get("path").(string))
}

func (c CfBuildpackResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	if d.Id() != "" {
		d, err := client.Finder().GetBuildpackFromCf(d.Id())
		if err != nil {
			return false, err
		}
		return d.GUID != "", nil
	}
	name := d.Get("name").(string)
	buildpack, err := client.Buildpack().FindByName(name)
	if err != nil {
		if _, ok := err.(*errors.ModelNotFoundError); ok {
			return false, nil
		}
		return false, err
	}
	d.SetId(buildpack.GUID)
	return true, nil
}
func (c CfBuildpackResource) generateFilename(buildpackPath string) (string, error) {
	if buildpackPath == "" {
		return "", nil
	}
	if common.IsWebURL(buildpackPath) {
		return path.Base(buildpackPath), nil
	}
	buildpackFileName := filepath.Base(buildpackPath)
	dir, err := filepath.Abs(buildpackPath)
	if err != nil {
		return "", err
	}
	buildpackFileName = filepath.Base(dir)
	stats, err := os.Stat(dir)
	if err != nil {
		return "", err
	}

	if stats.IsDir() {
		buildpackFileName += ".zip" // FIXME: remove once #71167394 is fixed
	}
	return buildpackFileName, nil
}
func (c CfBuildpackResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	name := d.Get("name").(string)
	buildpack, err := client.Finder().GetBuildpackFromCf(d.Id())
	if err != nil {
		return err
	}
	if buildpack.GUID == "" {
		log.Printf(
			"[WARN] removing buildpack %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			name,
		)
		d.SetId("")
		return nil
	}
	d.Set("name", buildpack.Name)
	bp, err := c.resourceObject(d)
	if err != nil {
		return err
	}
	if c.isSystemBuildpackManaged(bp) {
		return nil
	}
	if bp.Filename != buildpack.Filename && d.Get("path").(string) != "" {
		d.Set("path", buildpack.Filename)
	}
	d.Set("position", *buildpack.Position)
	d.Set("enabled", *buildpack.Enabled)
	d.Set("locked", *buildpack.Locked)
	return nil

}
func (c CfBuildpackResource) isSystemBuildpackManaged(buildpack models.Buildpack) bool {
	if buildpack.Filename == "" && *buildpack.Position == 1 && *buildpack.Enabled == true && *buildpack.Locked == false {
		return true
	}
	return false
}
func (c CfBuildpackResource) Update(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	name := d.Get("name").(string)
	buildpack, err := c.resourceObject(d)
	if err != nil {
		return err
	}
	buildpackCf, err := client.Finder().GetBuildpackFromCf(d.Id())
	if err != nil {
		return err
	}
	if buildpackCf.GUID == "" {
		log.Printf(
			"[WARN] removing organization %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			name,
		)
		d.SetId("")
		return nil
	}
	return c.updateBuildpack(client, buildpackCf, buildpack, d.Get("path").(string))
}
func (c CfBuildpackResource) updateBuildpack(client cf_client.Client, buildpackFrom, buildpackTo models.Buildpack, buildpackPath string) error {
	var err error
	if buildpackTo.Locked != buildpackFrom.Locked ||
		buildpackTo.Enabled != buildpackFrom.Enabled ||
		buildpackTo.Name != buildpackFrom.Name ||
		buildpackTo.Position != buildpackFrom.Position {

		_, err = client.Buildpack().Update(buildpackTo)
		if err != nil {
			return err
		}
	}
	if buildpackTo.Filename == "" {
		return nil
	}
	if buildpackTo.Filename != buildpackFrom.Filename {
		file, _, err := client.BuildpackBits().CreateBuildpackZipFile(buildpackPath)
		if err != nil {
			return err
		}
		err = client.BuildpackBits().UploadBuildpack(buildpackTo, file, buildpackTo.Filename)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c CfBuildpackResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	bp, err := c.resourceObject(d)
	if err != nil {
		return err
	}
	if c.isSystemBuildpackManaged(bp) {
		return nil
	}
	return client.Buildpack().Delete(d.Id())
}

func (c CfBuildpackResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"path": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"position": &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
			Default:  1,
		},
		"enabled": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
		},
		"locked": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
	}
}
func (c CfBuildpackResource) DataSourceSchema() map[string]*schema.Schema {
	return CreateDataSourceSchema(c)
}
func (c CfBuildpackResource) DataSourceRead(d *schema.ResourceData, meta interface{}) error {
	fn := CreateDataSourceReadFunc(c)
	return fn(d, meta)
}
