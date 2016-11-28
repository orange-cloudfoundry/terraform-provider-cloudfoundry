package resources

import (
	"github.com/hashicorp/terraform/helper/schema"
	"code.cloudfoundry.org/cli/cf/models"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"log"
	"reflect"
	"fmt"
	"bytes"
	"github.com/hashicorp/terraform/helper/hashcode"
	"strconv"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/resources/caching"
	"errors"
	"strings"
	"regexp"
)

var validProtocoles []string = []string{"icm", "tcp", "udp", "all"}

type CfSecurityGroupResource struct {
	CfResource
}

func NewCfSecurityGroupResource() CfResource {
	return &CfSecurityGroupResource{}
}

func (c CfSecurityGroupResource) resourceObject(d *schema.ResourceData) models.SecurityGroupFields {
	rulesSchema := d.Get("rules").(*schema.Set)
	rules := make([]map[string]interface{}, 0)
	for _, rule := range rulesSchema.List() {
		rules = append(rules, c.sanitizeRule(rule.(map[string]interface{})))
	}
	return models.SecurityGroupFields{
		GUID: d.Id(),
		Name: d.Get("name").(string),
		Rules: rules,
	}
}
func (c CfSecurityGroupResource) unSanitizeRule(rule map[string]interface{}) map[string]interface{} {
	unSanitizedRule := make(map[string]interface{})
	if _, ok := rule["code"]; !ok {
		unSanitizedRule["code"] = 0
	}
	if _, ok := rule["type"]; !ok {
		unSanitizedRule["type"] = 0
	}
	for index, content := range rule {
		unSanitizedRule[index] = content
	}
	return unSanitizedRule
}
func (c CfSecurityGroupResource) sanitizeRule(rule map[string]interface{}) map[string]interface{} {
	sanitizedRule := make(map[string]interface{})

	for index, content := range rule {
		if index == "code" && content.(int) == 0 {
			continue
		}
		if index == "type" && content.(int) == 0 {
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
	secGroup, err := c.getSecGroupFromCf(client, d.Id())
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
	secGroupCf, err := c.getSecGroupFromCf(client, d.Id())
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
	if d.Get("on_staging").(bool) != isOnStaging {
		err = c.updateBindingStaging(client, d.Id(), d.Get("on_staging").(bool))
		if err != nil {
			return err
		}
	}
	if d.Get("on_running").(bool) != isOnStaging {
		err = c.updateBindingStaging(client, d.Id(), d.Get("on_running").(bool))
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
		return false;
	}
	if rulesFrom == nil || rulesTo == nil {
		return true;
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
func (c CfSecurityGroupResource) getSecGroupFromCf(client cf_client.Client, secGroupId string) (models.SecurityGroupFields, error) {
	secGroups, err := caching.GetSecGroupsFromCf(client)
	if err != nil {
		return models.SecurityGroupFields{}, err
	}
	for _, secGroup := range secGroups {
		if secGroup.GUID == secGroupId {
			return secGroup.SecurityGroupFields, nil
		}
	}
	return models.SecurityGroupFields{}, nil
}
func (c CfSecurityGroupResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	name := d.Get("name").(string)
	secGroups, err := caching.GetSecGroupsFromCf(client)
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

			Elem:     &schema.Resource{
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
							match, _ := regexp.MatchString("^[0-9][0-9-,]+[^-,]$", ports)
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
					},
					"type": &schema.Schema{
						Type:     schema.TypeInt,
						Optional: true,
					},
					"log": &schema.Schema{
						Type:     schema.TypeBool,
						Default: true,
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

