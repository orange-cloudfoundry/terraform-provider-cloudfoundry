package resources

import (
	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/models"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"github.com/viant/toolbox"
	"log"
	"strings"
)

type CfDomainResource struct {
	CfResource
}

func (c CfDomainResource) resourceObject(d *schema.ResourceData) models.DomainFields {
	return models.DomainFields{
		GUID: d.Id(),
		Name: d.Get("name").(string),
		OwningOrganizationGUID: d.Get("org_owner_id").(string),
		Shared:                 d.Get("shared").(bool),
	}
}
func (c CfDomainResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	var err error
	domain := c.resourceObject(d)
	isShared := d.Get("shared").(bool)
	if ok, _ := c.Exists(d, meta); ok {
		log.Printf(
			"[INFO] skipping creation of domains %s/%s because it already exists on your Cloud Foundry",
			client.Config().ApiEndpoint,
			domain.Name,
		)
	} else {

		if isShared {
			err = c.createSharedDomain(client, domain, d.Get("router_group").(string))
		} else {
			err = c.createPrivateDomain(client, domain)
		}
		if err != nil {
			return err
		}
		c.Exists(d, meta)
	}
	domain.GUID = d.Id()
	domainCf, err := c.getDomainFromCf(client, domain)
	if err != nil {
		return err
	}
	orgsSchema := d.Get("orgs_shared_id").(*schema.Set)
	orgs := make([]string, 0)
	for _, org := range orgsSchema.List() {
		orgs = append(orgs, org.(string))
		err := client.Organizations().SharePrivateDomain(org.(string), d.Id())
		if err != nil {
			return err
		}
	}
	d.SetId(domainCf.GUID)
	if isShared {
		return nil
	}
	currentOrgs, err := c.getOrgsSharedIdFromCf(client, d.Id())
	if err != nil {
		return err
	}
	return c.updateSharedToOrg(client, domainCf, currentOrgs, orgs)
}
func (c CfDomainResource) getOrgsSharedIdFromCf(client cf_client.Client, domainGuid string) ([]string, error) {
	orgsId := make([]string, 0)
	orgs, err := client.Organizations().ListOrgs(0)
	if err != nil {
		return orgsId, err
	}
	for _, org := range orgs {
		err := client.Domain().ListDomainsForOrg(org.GUID, func(domainFound models.DomainFields) bool {
			if domainFound.GUID == domainGuid {
				orgsId = append(orgsId, org.GUID)
			}
			return true
		})
		if err != nil {
			return orgsId, err
		}
	}
	return orgsId, nil
}
func (c CfDomainResource) updateSharedToOrg(client cf_client.Client, domain models.DomainFields, currentOrgsId, wantedOrgsId []string) error {
	toCreate := make([]string, 0)
	toDelete := make([]string, 0)
	for _, orgId := range wantedOrgsId {
		if !toolbox.HasSliceAnyElements(currentOrgsId, orgId) {
			toCreate = append(toCreate, orgId)
		}
	}
	for _, orgId := range currentOrgsId {
		if !toolbox.HasSliceAnyElements(wantedOrgsId, orgId) && orgId != domain.OwningOrganizationGUID {
			toDelete = append(toDelete, orgId)
		}
	}
	for _, orgId := range toDelete {
		err := client.Organizations().UnsharePrivateDomain(orgId, domain.GUID)
		if err != nil {
			return err
		}
	}
	for _, orgId := range toCreate {
		err := client.Organizations().SharePrivateDomain(orgId, domain.GUID)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c CfDomainResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	domain := c.resourceObject(d)
	domainCf, err := c.getDomainFromCf(client, domain)
	if err != nil {
		return err
	}
	if domainCf.GUID == "" {
		log.Printf(
			"[WARN] removing domain %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			domain.Name,
		)
		d.SetId("")
		return nil
	}
	d.Set("name", domainCf.Name)
	d.Set("org_owner_id", domainCf.OwningOrganizationGUID)
	d.Set("shared", domainCf.Shared)
	d.Set("router_group", "")
	if domainCf.RouterGroupGUID != "" {
		err := client.RoutingAPI().ListRouterGroups(func(r models.RouterGroup) bool {
			if r.GUID == domainCf.RouterGroupGUID {
				d.Set("router_group", r.Name)
				return false
			}
			return true
		})
		if err != nil {
			return err
		}
	}
	orgsSharedSchema := schema.NewSet(d.Get("orgs_shared_id").(*schema.Set).F, make([]interface{}, 0))
	if domainCf.Shared {
		d.Set("orgs_shared_id", orgsSharedSchema)
		return nil
	}
	currentOrgs, err := c.getOrgsSharedIdFromCf(client, d.Id())
	if err != nil {
		return err
	}
	for _, orgId := range currentOrgs {
		if orgId == domainCf.OwningOrganizationGUID {
			continue
		}
		orgsSharedSchema.Add(orgId)
	}
	d.Set("orgs_shared_id", orgsSharedSchema)
	return nil

}
func (c CfDomainResource) createSharedDomain(client cf_client.Client, domain models.DomainFields, routerName string) error {
	var routerGuid string
	var err error
	if routerName == "" {
		routerGuid = ""
	} else {
		routerGuid, err = c.getRouterGuid(client, routerName)
		if err != nil {
			return err
		}
	}
	return client.Domain().CreateSharedDomain(domain.Name, routerGuid)
}
func (c CfDomainResource) createPrivateDomain(client cf_client.Client, domain models.DomainFields) error {
	if domain.OwningOrganizationGUID == "" {
		return fmt.Errorf("You need to set org_owner_id for the private domain '%s'.", domain.Name)
	}
	_, err := client.Domain().Create(domain.Name, domain.OwningOrganizationGUID)
	return err
}
func (c CfDomainResource) getDomainFromCf(client cf_client.Client, domain models.DomainFields) (models.DomainFields, error) {
	res := resources.DomainResource{}
	err := client.Gateways().CloudControllerGateway.GetResource(
		fmt.Sprintf("%s%s/%s",
			client.Config().ApiEndpoint,
			client.EndpointStrategy().PrivateDomainsURL(),
			domain.GUID,
		),
		&res)
	if err != nil {
		err = client.Gateways().CloudControllerGateway.GetResource(
			fmt.Sprintf("%s%s/%s",
				client.Config().ApiEndpoint,
				client.EndpointStrategy().SharedDomainsURL(),
				domain.GUID,
			),
			&res)
	}
	if err != nil {
		return models.DomainFields{}, err
	}
	return res.ToFields(), nil
}
func (c CfDomainResource) getRouterGuid(client cf_client.Client, routerName string) (string, error) {
	var router models.RouterGroup
	err := client.RoutingAPI().ListRouterGroups(func(r models.RouterGroup) bool {
		if r.Name == routerName {
			router = r
			return false
		}
		return true
	})
	if err != nil {
		return "", err
	}
	if router.GUID == "" {
		return "", fmt.Errorf("Can't found router group '%s' in Cloud Foundry.", routerName)
	}
	return router.GUID, nil
}
func (c CfDomainResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	domainName := d.Get("name").(string)
	var domain models.DomainFields
	var err error
	domain, err = client.Domain().FindSharedByName(domainName)
	if err != nil || domain.GUID == "" {
		domain, err = client.Domain().FindPrivateByName(domainName)
	}

	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	d.SetId(domain.GUID)
	return true, nil
}
func (c CfDomainResource) Update(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	domain := c.resourceObject(d)
	domainCf, err := c.getDomainFromCf(client, domain)
	if err != nil {
		return err
	}
	if domainCf.GUID == "" {
		log.Printf(
			"[WARN] removing domain %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			domain.Name,
		)
		d.SetId("")
		return nil
	}
	if !domainCf.Shared && domain.Shared == domainCf.Shared {
		orgsSchema := d.Get("orgs_shared_id").(*schema.Set)
		orgs := make([]string, 0)
		for _, org := range orgsSchema.List() {
			orgs = append(orgs, org.(string))
		}
		currentOrgs, err := c.getOrgsSharedIdFromCf(client, d.Id())
		if err != nil {
			return err
		}
		if err := c.updateSharedToOrg(client, domain, currentOrgs, orgs); err != nil {
			return err
		}
	}
	return nil
}
func (c CfDomainResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)

	err := client.Domain().DeleteSharedDomain(d.Id())
	if err != nil {
		return client.Domain().Delete(d.Id())
	}
	return nil
}
func (c CfDomainResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"router_group": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
		},
		"org_owner_id": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
		},
		"orgs_shared_id": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Set:      schema.HashString,
		},
		"shared": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
			ForceNew: true,
		},
	}
}
