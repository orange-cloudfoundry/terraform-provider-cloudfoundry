package resources

import (
	"bytes"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"fmt"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"log"
	"strconv"
)

var FLAGS map[string]bool = map[string]bool{
	"user_org_creation":                    false,
	"private_domain_creation":              true,
	"app_bits_upload":                      true,
	"app_scaling":                          true,
	"route_creation":                       true,
	"service_instance_creation":            true,
	"diego_docker":                         false,
	"set_roles_by_username":                true,
	"unset_roles_by_username":              true,
	"task_creation":                        false,
	"env_var_visibility":                   true,
	"space_scoped_private_broker_creation": true,
	"space_developer_env_var_visibility":   true,
}

type CfFeatureFlagsResource struct {
	CfResource
}

func (c CfFeatureFlagsResource) resourceObject(d *schema.ResourceData) map[string]bool {
	flags := make(map[string]bool)
	for flagName, _ := range FLAGS {
		flags[flagName] = d.Get(flagName).(bool)
	}
	customFlagsShema := d.Get("custom_flag").(*schema.Set)
	for _, elm := range customFlagsShema.List() {
		customFlag := elm.(map[string]interface{})
		flags[customFlag["name"].(string)] = customFlag["enabled"].(bool)
	}
	return flags
}
func (c CfFeatureFlagsResource) filteredFlagsWithCf(client cf_client.Client, wantedFlags map[string]bool) (map[string]bool, error) {
	flags := make(map[string]bool)
	flagsCfTmp, err := client.FeatureFlags().List()
	if err != nil {
		return flags, err
	}
	flagsCf := c.featureFlagsToMap(flagsCfTmp)
	for flagName, flagValue := range wantedFlags {
		if val, ok := flagsCf[flagName]; ok && val != flagValue {
			flags[flagName] = flagValue
		}
	}
	return flags, nil
}
func (c CfFeatureFlagsResource) featureFlagsToMap(flags []models.FeatureFlag) map[string]bool {
	flagsMap := make(map[string]bool)
	for _, flag := range flags {
		flagsMap[flag.Name] = flag.Enabled
	}
	return flagsMap
}
func (c CfFeatureFlagsResource) updateFlags(client cf_client.Client, flags map[string]bool) error {
	finalFlags, err := c.filteredFlagsWithCf(client, flags)
	if err != nil {
		return err
	}
	for flagName, flagValue := range finalFlags {
		err := client.FeatureFlags().Update(flagName, flagValue)
		if err != nil {
			if _, ok := err.(*errors.HTTPNotFoundError); ok {
				continue
			}
			return err
		}
	}
	return nil
}
func (c CfFeatureFlagsResource) generateId(flags map[string]bool) string {
	var buf bytes.Buffer
	for flagName, flagValue := range flags {
		buf.WriteString(fmt.Sprintf("%s-", flagName))
		buf.WriteString(fmt.Sprintf("%s-", strconv.FormatBool(flagValue)))
	}
	return strconv.Itoa(hashcode.String(buf.String()))
}
func (c CfFeatureFlagsResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	flags := c.resourceObject(d)

	if ok, _ := c.Exists(d, meta); ok {
		log.Printf(
			"[INFO] skipping creation of flags %s because they already exists on your Cloud Foundry",
			client.Config().ApiEndpoint,
		)
		return nil
	}
	err := c.updateFlags(client, flags)
	if err != nil {
		return err
	}
	d.SetId(c.generateId(flags))
	return nil
}

func (c CfFeatureFlagsResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	flags := c.resourceObject(d)
	flagsCfTmp, err := client.FeatureFlags().List()
	if err != nil {
		return err
	}
	customFlagMap := make(map[string]bool, len(flags))
	for k, v := range flags {
		customFlagMap[k] = v
	}
	flagsCf := c.featureFlagsToMap(flagsCfTmp)
	for flagName, _ := range FLAGS {
		delete(customFlagMap, flagName)
		if val, ok := flagsCf[flagName]; ok && val != flags[flagName] {
			d.Set(flagName, val)
		}
	}
	customFlagSchema := schema.NewSet(d.Get("custom_flag").(*schema.Set).F, make([]interface{}, 0))
	for flagName, flagValue := range customFlagMap {
		m := map[string]interface{}{
			"name":    flagName,
			"enabled": flagValue,
		}
		if val, ok := flagsCf[flagName]; ok && val != flagValue {
			m["enabled"] = val
		}
		customFlagSchema.Add(m)
	}
	d.Set("custom_flag", customFlagSchema)
	return nil
}
func (c CfFeatureFlagsResource) Update(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	flags := c.resourceObject(d)
	err := c.updateFlags(client, flags)
	if err != nil {
		return err
	}
	return nil
}

func (c CfFeatureFlagsResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	return c.updateFlags(client, FLAGS)
}
func (c CfFeatureFlagsResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	if d.Id() != "" {
		return true, nil
	}
	flags := c.resourceObject(d)
	finalFlags, err := c.filteredFlagsWithCf(client, flags)
	if err != nil {
		return false, err
	}
	exist := len(finalFlags) == 0
	if exist {
		d.SetId(c.generateId(flags))
	}
	return exist, nil
}
func (c CfFeatureFlagsResource) Schema() map[string]*schema.Schema {
	flagsSchema := make(map[string]*schema.Schema)
	for flagName, defaultValue := range FLAGS {
		flagsSchema[flagName] = &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
			Default:  defaultValue,
		}
	}
	flagsSchema["custom_flag"] = &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,

		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
				"enabled": &schema.Schema{
					Type:     schema.TypeBool,
					Required: true,
				},
			},
		},
		Set: func(v interface{}) int {
			var buf bytes.Buffer
			m := v.(map[string]interface{})
			buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
			buf.WriteString(fmt.Sprintf("%s-", strconv.FormatBool(m["enabled"].(bool))))
			return hashcode.String(buf.String())
		},
	}
	return flagsSchema
}
