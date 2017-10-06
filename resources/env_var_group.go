package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"log"
	"strconv"
)

type CfEnvVarGroupResource struct{}
type EnvVarGroupMap struct {
	Running map[string]string
	Staging map[string]string
}

func (e EnvVarGroupMap) ToSchema(d *schema.ResourceData) *schema.Set {
	schemaEnvVar := schema.NewSet(d.Get("env_var").(*schema.Set).F, make([]interface{}, 0))
	mapInterface := make(map[string]map[string]interface{})
	for key, value := range e.Running {
		m := map[string]interface{}{
			"key":     key,
			"value":   value,
			"running": true,
			"staging": false,
		}
		mapInterface[key] = m
	}
	for key, value := range e.Staging {
		if _, ok := mapInterface[key]; ok {
			mapInterface[key]["staging"] = true
			continue
		}
		m := map[string]interface{}{
			"key":     key,
			"value":   value,
			"running": false,
			"staging": true,
		}
		mapInterface[key] = m
	}
	for _, data := range mapInterface {
		schemaEnvVar.Add(data)
	}
	return schemaEnvVar
}
func (e EnvVarGroupMap) ToJsonRunning() string {
	b, _ := json.Marshal(e.Running)
	return string(b)
}
func (e EnvVarGroupMap) ToJsonStaging() string {
	b, _ := json.Marshal(e.Staging)
	return string(b)
}
func (c CfEnvVarGroupResource) resourceObject(d *schema.ResourceData) EnvVarGroupMap {
	runningMap := make(map[string]string)
	stagingMap := make(map[string]string)
	envVarShema := d.Get("env_var").(*schema.Set)
	for _, elm := range envVarShema.List() {
		envVar := elm.(map[string]interface{})
		if envVar["running"].(bool) {
			runningMap[envVar["key"].(string)] = envVar["value"].(string)
		}
		if envVar["staging"].(bool) {
			stagingMap[envVar["key"].(string)] = envVar["value"].(string)
		}
	}
	return EnvVarGroupMap{
		Running: runningMap,
		Staging: stagingMap,
	}
}

func (c CfEnvVarGroupResource) generateId(evg EnvVarGroupMap) string {
	var buf bytes.Buffer
	for key, value := range evg.Staging {
		buf.WriteString(fmt.Sprintf("%s-", key))
		buf.WriteString(fmt.Sprintf("%s-", value))
	}
	for key, value := range evg.Running {
		buf.WriteString(fmt.Sprintf("%s-", key))
		buf.WriteString(fmt.Sprintf("%s-", value))
	}
	return strconv.Itoa(hashcode.String(buf.String()))
}
func (c CfEnvVarGroupResource) retrieveEnvVarGroupMapFromCf(client cf_client.Client) (EnvVarGroupMap, error) {
	envVarRunning, err := client.EnvVarGroup().ListRunning()
	if err != nil {
		return EnvVarGroupMap{}, err
	}
	envVarStaging, err := client.EnvVarGroup().ListStaging()
	if err != nil {
		return EnvVarGroupMap{}, err
	}
	runningMap := make(map[string]string)
	stagingMap := make(map[string]string)
	for _, envVar := range envVarRunning {
		runningMap[envVar.Name] = envVar.Value
	}
	for _, envVar := range envVarStaging {
		stagingMap[envVar.Name] = envVar.Value
	}
	return EnvVarGroupMap{
		Running: runningMap,
		Staging: stagingMap,
	}, nil
}
func (c CfEnvVarGroupResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	envVars := c.resourceObject(d)

	if ok, _ := c.Exists(d, meta); ok {
		log.Printf(
			"[INFO] skipping creation of env vars %s because they already exists on your Cloud Foundry",
			client.Config().ApiEndpoint,
		)
		return nil
	}
	err := c.updateEnvVar(client, envVars)
	if err != nil {
		return err
	}
	d.SetId(c.generateId(envVars))
	return nil
}
func (c CfEnvVarGroupResource) updateEnvVar(client cf_client.Client, envVar EnvVarGroupMap) error {
	err := client.EnvVarGroup().SetStaging(envVar.ToJsonStaging())
	if err != nil {
		return err
	}
	return client.EnvVarGroup().SetRunning(envVar.ToJsonRunning())
}
func (c CfEnvVarGroupResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	envVarsCf, err := c.retrieveEnvVarGroupMapFromCf(client)
	if err != nil {
		return err
	}
	d.Set("env_var", envVarsCf.ToSchema(d))
	return nil
}
func (c CfEnvVarGroupResource) Update(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	envVars := c.resourceObject(d)
	err := c.updateEnvVar(client, envVars)
	if err != nil {
		return err
	}
	d.SetId(c.generateId(envVars))
	return nil
}

func (c CfEnvVarGroupResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	err := client.EnvVarGroup().SetRunning("{}")
	if err != nil {
		return err
	}
	return client.EnvVarGroup().SetStaging("{}")
}
func (c CfEnvVarGroupResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	if d.Id() != "" {
		return true, nil
	}
	envVars := c.resourceObject(d)
	envVarsCf, err := c.retrieveEnvVarGroupMapFromCf(client)
	if err != nil {
		return false, err
	}
	diff := envVars.ToSchema(d).Difference(envVarsCf.ToSchema(d))
	exist := (diff == nil || diff.Len() == 0)
	if exist {
		d.SetId(c.generateId(envVars))
	}
	return exist, nil
}
func (c CfEnvVarGroupResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"env_var": &schema.Schema{
			Type:     schema.TypeSet,
			Required: true,

			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"key": &schema.Schema{
						Type:     schema.TypeString,
						Required: true,
					},
					"value": &schema.Schema{
						Type:      schema.TypeString,
						Required:  true,
						Sensitive: true,
					},
					"running": &schema.Schema{
						Type:     schema.TypeBool,
						Required: true,
					},
					"staging": &schema.Schema{
						Type:     schema.TypeBool,
						Required: true,
					},
				},
			},
			Set: func(v interface{}) int {
				var buf bytes.Buffer
				m := v.(map[string]interface{})
				buf.WriteString(fmt.Sprintf("%s-", m["key"].(string)))
				buf.WriteString(fmt.Sprintf("%s-", m["value"].(string)))
				buf.WriteString(fmt.Sprintf("%s-", strconv.FormatBool(m["running"].(bool))))
				buf.WriteString(fmt.Sprintf("%s-", strconv.FormatBool(m["staging"].(bool))))
				return hashcode.String(buf.String())
			},
		},
	}
}
