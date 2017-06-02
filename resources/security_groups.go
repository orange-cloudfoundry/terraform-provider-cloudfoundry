package resources

import (
	"bytes"
	"code.cloudfoundry.org/cli/cf/models"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var validProtocoles []string = []string{"icmp", "tcp", "udp", "all"}

type CfSecurityGroupResource struct {
	CfResource
}

func (c CfSecurityGroupResource) resourceObject(d *schema.ResourceData) models.SecurityGroupFields {
	rulesSchema := d.Get("rules").(*schema.Set)
	rules := make([]map[string]interface{}, 0)
	for _, rule := range rulesSchema.List() {
		rules = append(rules, c.sanitizeRule(rule.(map[string]interface{})))
	}
	return models.SecurityGroupFields{
		GUID:  d.Id(),
		Name:  d.Get("name").(string),
		Rules: rules,
	}
}
func (c CfSecurityGroupResource) unSanitizeRule(rule map[string]interface{}) map[string]interface{} {
	unSanitizedRule := make(map[string]interface{})
	if _, ok := rule["code"]; !ok {
		unSanitizedRule["code"] = -1
	} else {
		rule["code"] = c.convertRuleParamFloatToInt(rule["code"])
	}
	if _, ok := rule["log"]; !ok {
		unSanitizedRule["log"] = false
	}
	if _, ok := rule["type"]; !ok {
		unSanitizedRule["type"] = -1
	} else {
		rule["type"] = c.convertRuleParamFloatToInt(rule["type"])
	}
	if _, ok := rule["ports"]; !ok {
		unSanitizedRule["ports"] = ""
	}
	if _, ok := rule["destination"]; !ok {
		unSanitizedRule["destination"] = ""
	}
	if _, ok := rule["description"]; !ok {
		unSanitizedRule["description"] = ""
	}
	for index, content := range rule {
		unSanitizedRule[index] = content
	}
	return unSanitizedRule
}
func (c CfSecurityGroupResource) convertRuleParamFloatToInt(param interface{}) int {
	kindParam := reflect.TypeOf(param).Kind()
	if kindParam == reflect.Float32 {
		return int(param.(float32))
	}
	if kindParam == reflect.Float64 {
		return int(param.(float64))
	}
	return param.(int)
}
func (c CfSecurityGroupResource) sanitizeRule(rule map[string]interface{}) map[string]interface{} {
	sanitizedRule := make(map[string]interface{})

	for index, content := range rule {
		if index == "code" && content.(int) == -1 {
			continue
		}
		if index == "log" && content.(bool) == false {
			continue
		}
		if index == "type" && content.(int) == -1 {
			continue
		}
		if index == "ports" && content.(string) == "" {
			continue
		}
		if index == "destination" && content.(string) == "" {
			continue
		}
		if index == "description" && content.(string) == "" {
			continue
		}
		sanitizedRule[index] = content
	}
	return sanitizedRule
}
func (c CfSecurityGroupResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	secGroup := c.resourceObject(d)
	var err error
	if ok, _ := c.Exists(d, meta); ok {
		log.Printf(
			"[INFO] skipping creation of security group %s/%s because it already exists on your Cloud Foundry",
			client.Config().ApiEndpoint,
			secGroup.Name,
		)
		err = client.SecurityGroups().Update(d.Id(), secGroup.Rules)
		if err != nil {
			return err
		}
	} else {
		err = client.SecurityGroups().Create(secGroup.Name, secGroup.Rules)
		if err != nil {
			return err
		}
		_, err = c.Exists(d, meta)
		if err != nil {
			return err
		}
	}
	if d.Get("on_staging").(bool) {
		err = client.SecurityGroupsStagingBinder().BindToStagingSet(d.Id())
		if err != nil {
			return err
		}
	}
	if d.Get("on_running").(bool) {
		err = client.SecurityGroupsRunningBinder().BindToRunningSet(d.Id())
		if err != nil {
			return err
		}
	}
	return nil
}

func (c CfSecurityGroupResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	secGroupName := d.Get("name").(string)
	secGroup, err := client.Finder().GetSecGroupFromCf(d.Id())
	if err != nil {
		return err
	}
	if secGroup.GUID == "" {
		log.Printf(
			"[WARN] removing security group %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			secGroupName,
		)
		d.SetId("")
		return nil
	}
	d.Set("name", secGroup.Name)
	rules := make([]interface{}, 0)
	rulesSchema := schema.NewSet(d.Get("rules").(*schema.Set).F, rules)
	for _, rule := range secGroup.Rules {
		rulesSchema.Add(c.unSanitizeRule(rule))
	}
	d.Set("rules", rulesSchema)
	isOnStaging, err := c.isOnStaging(client, secGroup.GUID)
	if err != nil {
		return err
	}
	isOnRunning, err := c.isOnRunning(client, secGroup.GUID)
	if err != nil {
		return err
	}
	d.Set("on_staging", isOnStaging)
	d.Set("on_running", isOnRunning)
	return nil
}
func (c CfSecurityGroupResource) Update(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	secGroup := c.resourceObject(d)
	secGroupCf, err := client.Finder().GetSecGroupFromCf(d.Id())
	if err != nil {
		return err
	}
	if secGroupCf.GUID == "" {
		log.Printf(
			"[WARN] removing security group %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			secGroup.Name,
		)
		d.SetId("")
		return nil
	}
	if c.isRulesChange(secGroupCf.Rules, secGroup.Rules) {
		client.SecurityGroups().Update(d.Id(), secGroup.Rules)
	}
	isOnStaging, err := c.isOnStaging(client, d.Id())
	if err != nil {
		return err
	}
	isOnRunning, err := c.isOnRunning(client, d.Id())
	if err != nil {
		return err
	}
	if d.Get("on_staging").(bool) != isOnStaging {
		err = c.updateBindingStaging(client, d.Id(), d.Get("on_staging").(bool))
		if err != nil {
			return err
		}
	}
	if d.Get("on_running").(bool) != isOnRunning {
		err = c.updateBindingRunning(client, d.Id(), d.Get("on_running").(bool))
		if err != nil {
			return err
		}
	}
	return nil
}
func (c CfSecurityGroupResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	return client.SecurityGroups().Delete(d.Id())
}
func (c CfSecurityGroupResource) updateBindingStaging(client cf_client.Client, guid string, onStaging bool) error {
	if onStaging {
		return client.SecurityGroupsStagingBinder().BindToStagingSet(guid)
	}
	return client.SecurityGroupsStagingBinder().UnbindFromStagingSet(guid)
}
func (c CfSecurityGroupResource) updateBindingRunning(client cf_client.Client, guid string, onRunning bool) error {
	if onRunning {
		return client.SecurityGroupsRunningBinder().BindToRunningSet(guid)
	}
	return client.SecurityGroupsRunningBinder().UnbindFromRunningSet(guid)
}
func (c CfSecurityGroupResource) isRulesChange(rulesFrom, rulesTo []map[string]interface{}) bool {
	if rulesFrom == nil && rulesTo == nil {
		return false
	}
	if rulesFrom == nil || rulesTo == nil {
		return true
	}
	if len(rulesFrom) != len(rulesTo) {
		return true
	}
	for i := range rulesFrom {
		if !reflect.DeepEqual(rulesFrom[i], rulesTo[i]) {
			return true
		}
	}
	for i := range rulesTo {
		if !reflect.DeepEqual(rulesFrom[i], rulesTo[i]) {
			return true
		}
	}
	return false

}
func (c CfSecurityGroupResource) isOnStaging(client cf_client.Client, secGroupId string) (bool, error) {
	secGroups, err := client.SecurityGroupsStagingBinder().List()
	if err != nil {
		return false, err
	}
	return c.existsSecurityGroup(secGroups, secGroupId), nil
}
func (c CfSecurityGroupResource) isOnRunning(client cf_client.Client, secGroupId string) (bool, error) {
	secGroups, err := client.SecurityGroupsRunningBinder().List()
	if err != nil {
		return false, err
	}
	return c.existsSecurityGroup(secGroups, secGroupId), nil
}
func (c CfSecurityGroupResource) existsSecurityGroup(secGroups []models.SecurityGroupFields, secGroupId string) bool {
	for _, secGroup := range secGroups {
		if secGroup.GUID == secGroupId {
			return true
		}
	}
	return false
}
func (c CfSecurityGroupResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	if d.Id() != "" {
		d, err := client.Finder().GetSecGroupFromCf(d.Id())
		if err != nil {
			return false, err
		}
		return d.GUID != "", nil
	}
	name := d.Get("name").(string)
	secGroups, err := client.SecurityGroups().FindAll()
	if err != nil {
		return false, err
	}
	for _, secGroup := range secGroups {
		if secGroup.Name == name {
			d.SetId(secGroup.GUID)
			return true, nil
		}
	}

	return false, nil
}

func (c CfSecurityGroupResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"rules": &schema.Schema{
			Type:     schema.TypeSet,
			Required: true,

			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"protocol": &schema.Schema{
						Type:     schema.TypeString,
						Required: true,
						ValidateFunc: func(elem interface{}, index string) ([]string, []error) {
							prot := elem.(string)
							found := false
							for _, validProt := range validProtocoles {
								if validProt == prot {
									found = true
									break
								}
							}
							if found {
								return make([]string, 0), make([]error, 0)
							}
							errMsg := fmt.Sprintf(
								"Protocol '%s' is not valid, it must be one of %s",
								prot,
								strings.Join(validProtocoles, ", "),
							)
							err := errors.New(errMsg)
							return make([]string, 0), []error{err}
						},
					},
					"destination": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
					},
					"description": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
					},
					"ports": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
						ValidateFunc: func(elem interface{}, index string) ([]string, []error) {
							ports := elem.(string)
							match, _ := regexp.MatchString("^[0-9][0-9-,]*[0-9]?$", ports)
							if match {
								return make([]string, 0), make([]error, 0)
							}
							errMsg := fmt.Sprintf(
								"Ports '%s' is not valid. (valid examples: '443', '80,8080,8081', '8080-8081')",
								ports,
							)
							err := errors.New(errMsg)
							return make([]string, 0), []error{err}
						},
					},
					"code": &schema.Schema{
						Type:     schema.TypeInt,
						Optional: true,
						Default:  -1,
					},
					"type": &schema.Schema{
						Type:     schema.TypeInt,
						Optional: true,
						Default:  -1,
					},
					"log": &schema.Schema{
						Type:     schema.TypeBool,
						Default:  false,
						Optional: true,
					},
				},
			},
			Set: func(v interface{}) int {
				var buf bytes.Buffer
				m := v.(map[string]interface{})
				buf.WriteString(fmt.Sprintf("%s-", m["protocol"].(string)))
				buf.WriteString(fmt.Sprintf("%s-", m["destination"].(string)))
				buf.WriteString(fmt.Sprintf("%s-", m["description"].(string)))
				buf.WriteString(fmt.Sprintf("%s-", m["ports"].(string)))
				buf.WriteString(fmt.Sprintf("%s-", strconv.Itoa(m["code"].(int))))
				buf.WriteString(fmt.Sprintf("%s-", strconv.Itoa(m["type"].(int))))
				buf.WriteString(fmt.Sprintf("%s-", strconv.FormatBool(m["log"].(bool))))
				return hashcode.String(buf.String())
			},
		},
		"on_staging": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		"on_running": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
	}
}
func (c CfSecurityGroupResource) DataSourceSchema() map[string]*schema.Schema {
	return CreateDataSourceSchema(c)
}
func (c CfSecurityGroupResource) DataSourceRead(d *schema.ResourceData, meta interface{}) error {
	fn := CreateDataSourceReadFunc(c)
	return fn(d, meta)
}
