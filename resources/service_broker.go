package resources

import (
	"github.com/hashicorp/terraform/helper/schema"
	"bytes"
	"fmt"
	"github.com/hashicorp/terraform/helper/hashcode"
	"code.cloudfoundry.org/cli/cf/models"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"strings"
	"log"
	"errors"
)

type CfServiceBrokerResource struct {
	CfResource
}
type ServiceAccess struct {
	Service string
	Plan    string
	OrgId   string
}

func NewCfServiceBrokerResource() CfResource {
	return &CfServiceBrokerResource{}
}
func (c CfServiceBrokerResource) serviceAccessObjects(d *schema.ResourceData) []ServiceAccess {
	serviceAccessShema := d.Get("service_access").(*schema.Set)
	serviceAccesses := make([]ServiceAccess, 0)
	for _, serviceAccess := range serviceAccessShema.List() {
		serviceAccesses = append(
			serviceAccesses,
			c.serviceAccessObject(serviceAccess.(map[string]interface{})),
		)
	}
	return serviceAccesses
}
func (c CfServiceBrokerResource) serviceAccessObject(serviceAccessMap map[string]interface{}) ServiceAccess {
	return ServiceAccess{
		Service: serviceAccessMap["service"].(string),
		Plan: serviceAccessMap["plan"].(string),
		OrgId: serviceAccessMap["org_id"].(string),
	}
}
func (c CfServiceBrokerResource) getFullServicesAccessDef(client cf_client.Client, serviceBroker models.ServiceBroker, servicesAccess []ServiceAccess) ([]ServiceAccess, error) {
	fullServicesAccess := make([]ServiceAccess, 0)
	for _, serviceAccess := range servicesAccess {
		servicesAccessTemp, err := c.getFullServiceAccessDef(client, serviceBroker, serviceAccess)
		if err != nil {
			return make([]ServiceAccess, 0), err
		}
		fullServicesAccess = append(fullServicesAccess, servicesAccessTemp...)
	}
	return fullServicesAccess, nil
}
func (c CfServiceBrokerResource) getFullServiceAccessDef(client cf_client.Client, serviceBroker models.ServiceBroker, serviceAccess ServiceAccess) ([]ServiceAccess, error) {
	servicesAccess := make([]ServiceAccess, 0)
	var err error
	service := c.getService(serviceBroker, serviceAccess.Service)
	if service.GUID == "" {
		return servicesAccess, errors.New(fmt.Sprintf("Service '%s' doesn't exist in broker '%s'.",
			serviceAccess.Service, serviceBroker.Name))
	}
	if serviceAccess.OrgId != "" && serviceAccess.Plan != "" {
		servicesAccess = append(servicesAccess, serviceAccess)
	}
	if serviceAccess.OrgId != "" {
		servicesAccess = c.getServicesAccessDefWithOnlyOrg(service, serviceAccess)
	}
	if serviceAccess.Plan != "" {
		servicesAccess, err = c.getServicesAccessDefWithOnlyPlan(client, service, serviceAccess)
	}
	if serviceAccess.OrgId == "" && serviceAccess.Plan == "" {
		servicesAccess, err = c.getServicesAccessDefWithoutPlanAndOrg(client, service, serviceAccess)
	}
	if err != nil {
		return make([]ServiceAccess, 0), err
	}
	return servicesAccess, err
}
func (c CfServiceBrokerResource) resourceObject(d *schema.ResourceData, meta interface{}) (models.ServiceBroker, error) {
	client := meta.(cf_client.Client)
	password, err := client.Decrypter().Decrypt(d.Get("password").(string))
	if err != nil {
		return models.ServiceBroker{}, err
	}
	return models.ServiceBroker{
		GUID: d.Id(),
		Name: d.Get("name").(string),
		URL: d.Get("url").(string),
		Username: d.Get("username").(string),
		Password: password,
	}, nil
}
func (c CfServiceBrokerResource) transformServicesAccessToMap(serviceAccess ServiceAccess) map[string]interface{} {
	return map[string]interface{}{
		"service": serviceAccess.Service,
		"plan": serviceAccess.Plan,
		"org_id": serviceAccess.OrgId,
	}
}
func (c CfServiceBrokerResource) retrieveServicesAccessFromBroker(client cf_client.Client, serviceBroker models.ServiceBroker) ([]ServiceAccess, error) {
	servicesAccess := make([]ServiceAccess, 0)

	for _, service := range serviceBroker.Services {
		servicesAccessInAllOrgsTemp := make([]ServiceAccess, 0)
		servicesAccessOrgAndPlanTemp := make([]ServiceAccess, 0)
		haveAllPlanInAllOrg := true
		for _, plan := range service.Plans {
			isPlanInAllOrg, err := c.isPlanInAllOrgs(client, plan.GUID)
			if err != nil {
				return servicesAccess, err
			}
			if isPlanInAllOrg {
				servicesAccessInAllOrgsTemp = append(servicesAccessInAllOrgsTemp, ServiceAccess{
					Service: service.Label,
					Plan: plan.Name,
				})
				continue
			}
			haveAllPlanInAllOrg = false
			visibilities, err := c.getPlanVisibilitiesForPlan(client, plan.GUID)
			if err != nil {
				return servicesAccess, err
			}
			for _, visibility := range visibilities {
				servicesAccessOrgAndPlanTemp = append(servicesAccessOrgAndPlanTemp, ServiceAccess{
					Service: service.Label,
					Plan: plan.Name,
					OrgId: visibility.OrganizationGUID,
				})
			}

		}
		if haveAllPlanInAllOrg {
			servicesAccess = append(servicesAccess, ServiceAccess{
				Service: service.Label,
			})
			continue
		}
		servicesAccess = append(servicesAccess, servicesAccessInAllOrgsTemp...)
		onlyWithOrg, fullServiceAccess := c.splitServiceAccess(servicesAccessOrgAndPlanTemp, len(service.Plans))
		servicesAccess = append(servicesAccess, onlyWithOrg...)
		servicesAccess = append(servicesAccess, fullServiceAccess...)
	}

	return servicesAccess, nil
}
func (c CfServiceBrokerResource) getPlanVisibilitiesForPlan(client cf_client.Client, planId string) ([]models.ServicePlanVisibilityFields, error) {
	return client.ServicePlanVisibilities().Search(map[string]string{"service_plan_guid": planId})
}
func (c CfServiceBrokerResource) getPlanVisibilityForPlanAndOrg(client cf_client.Client, planId, orgId string) (models.ServicePlanVisibilityFields, error) {
	visibilities, err := client.ServicePlanVisibilities().Search(map[string]string{
		"service_plan_guid": planId,
		"organization_guid": orgId,
	})
	if err != nil {
		return models.ServicePlanVisibilityFields{}, err
	}
	if len(visibilities) == 0 {
		return models.ServicePlanVisibilityFields{}, nil
	}
	return visibilities[0], nil
}
func (c CfServiceBrokerResource) splitServiceAccess(servicesAccess []ServiceAccess, numberPlan int) (onlyWithOrg []ServiceAccess, full []ServiceAccess) {
	onlyWithOrg = make([]ServiceAccess, 0)
	full = make([]ServiceAccess, 0)
	orgs := make(map[string]bool)
	for _, serviceAccess := range servicesAccess {
		if orgs[serviceAccess.OrgId + serviceAccess.Plan] {
			continue
		}
		if len(c.getServicesAccessForOrg(servicesAccess, serviceAccess.OrgId)) == numberPlan {
			onlyWithOrg = append(onlyWithOrg, ServiceAccess{
				Service: serviceAccess.Service,
				OrgId: serviceAccess.OrgId,
			})
			orgs[serviceAccess.OrgId + serviceAccess.Plan] = true
			continue
		}
		full = append(full, serviceAccess)
	}
	return
}
func (c CfServiceBrokerResource) getServicesAccessForOrg(servicesAccess []ServiceAccess, orgId string) []ServiceAccess {
	servicesAccessInOrg := make([]ServiceAccess, 0)
	for _, serviceAccess := range servicesAccess {
		if serviceAccess.OrgId == orgId {
			servicesAccessInOrg = append(servicesAccessInOrg, serviceAccess)
		}
	}
	return servicesAccessInOrg
}
func (c CfServiceBrokerResource) isPlanInOrg(client cf_client.Client, planGuid string, orgGuid string) (bool, error) {
	visibility, err := c.getPlanVisibilityForPlanAndOrg(client, planGuid, orgGuid)
	if err != nil {
		return false, err
	}
	if visibility.GUID == "" {
		return false, nil
	}
	return true, nil
}
func (c CfServiceBrokerResource) isPlanInAllOrgs(client cf_client.Client, planGuid string) (bool, error) {
	orgs, err := client.Organizations().ListOrgs(0)
	if err != nil {
		return false, err
	}
	for _, org := range orgs {
		isInOrg, err := c.isPlanInOrg(client, planGuid, org.GUID)
		if err != nil {
			return false, err
		}
		if !isInOrg {
			return false, err
		}
	}
	return true, nil
}
func (c CfServiceBrokerResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	serviceBroker, err := c.resourceObject(d, meta)
	if err != nil {
		return err
	}
	if ok, _ := c.Exists(d, meta); ok {
		log.Printf(
			"[INFO] skipping creation of service broker %s/%s because it already exists on your Cloud Foundry",
			client.Config().ApiEndpoint,
			serviceBroker.Name,
		)
	} else {
		err := client.ServiceBrokers().Create(
			serviceBroker.Name,
			serviceBroker.URL,
			serviceBroker.Username,
			serviceBroker.Password,
			"",
		)
		if err != nil {
			return err
		}
		c.Exists(d, meta)
	}
	serviceBrokerCf, err := c.getServiceBrokerFromCf(client, d.Id())
	servicesAccess := c.serviceAccessObjects(d)
	return c.updateServicesAccess(client, serviceBrokerCf, servicesAccess)
}
func (c CfServiceBrokerResource) updateServicesAccess(client cf_client.Client, serviceBroker models.ServiceBroker, servicesAccess []ServiceAccess) error {
	for _, serviceAccess := range servicesAccess {
		err := c.updateServiceAccess(client, serviceBroker, serviceAccess)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c CfServiceBrokerResource) updateServiceAccess(client cf_client.Client, serviceBroker models.ServiceBroker, serviceAccess ServiceAccess) error {
	service := c.getService(serviceBroker, serviceAccess.Service)
	if service.GUID == "" {
		return errors.New(fmt.Sprintf("Service '%s' doesn't exist in broker '%s'.",
			serviceAccess.Service, serviceBroker.Name))
	}
	if serviceAccess.OrgId != "" && serviceAccess.Plan != "" {
		return c.updateServiceAccessWithPlanAndOrg(client, service, serviceAccess)
	}
	var servicesAccess []ServiceAccess
	var err error
	if serviceAccess.OrgId != "" {
		servicesAccess = c.getServicesAccessDefWithOnlyOrg(service, serviceAccess)
	}
	if serviceAccess.Plan != "" {
		servicesAccess, err = c.getServicesAccessDefWithOnlyPlan(client, service, serviceAccess)
	}
	if serviceAccess.OrgId == "" && serviceAccess.Plan == "" {
		servicesAccess, err = c.getServicesAccessDefWithoutPlanAndOrg(client, service, serviceAccess)
	}
	if err != nil {
		return err
	}
	for _, serviceAccessToUpdate := range servicesAccess {
		err = c.updateServiceAccessWithPlanAndOrg(client, service, serviceAccessToUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c CfServiceBrokerResource) getServicesAccessDefWithOnlyOrg(service models.ServiceOffering, serviceAccess ServiceAccess) ([]ServiceAccess) {
	servicesAccess := make([]ServiceAccess, 0)
	for _, plan := range service.Plans {
		servicesAccess = append(servicesAccess, ServiceAccess{
			Service: serviceAccess.Service,
			OrgId: serviceAccess.OrgId,
			Plan: plan.Name,
		})
	}
	return servicesAccess
}
func (c CfServiceBrokerResource) getServicesAccessDefWithOnlyPlan(client cf_client.Client, service models.ServiceOffering, serviceAccess ServiceAccess) ([]ServiceAccess, error) {
	servicesAccess := make([]ServiceAccess, 0)
	orgs, err := client.Organizations().ListOrgs(0)
	if err != nil {
		return servicesAccess, err
	}
	for _, org := range orgs {
		servicesAccess = append(servicesAccess, ServiceAccess{
			Service: serviceAccess.Service,
			OrgId: org.GUID,
			Plan: serviceAccess.Plan,
		})
	}
	return servicesAccess, nil
}
func (c CfServiceBrokerResource) getServicesAccessDefWithoutPlanAndOrg(client cf_client.Client, service models.ServiceOffering, serviceAccess ServiceAccess) ([]ServiceAccess, error) {
	servicesAccess := make([]ServiceAccess, 0)
	for _, plan := range service.Plans {
		newServicesAccess, err := c.getServicesAccessDefWithOnlyPlan(client, service, ServiceAccess{
			Service: serviceAccess.Service,
			Plan: plan.Name,
		})
		if err != nil {
			return servicesAccess, err
		}
		servicesAccess = append(servicesAccess, newServicesAccess...)
	}
	return servicesAccess, nil
}
func (c CfServiceBrokerResource) updateServiceAccessWithPlanAndOrg(client cf_client.Client, service models.ServiceOffering, serviceAccess ServiceAccess) error {
	plan := c.getServicePlan(service, serviceAccess.Plan)
	if plan.GUID == "" {
		return errors.New(fmt.Sprintf("Plan '%s' doesn't exist in service '%s'.",
			serviceAccess.Service, serviceAccess.Plan))
	}
	err := client.ServicePlanVisibilities().Create(plan.GUID, serviceAccess.OrgId)
	if err != nil {
		if strings.Contains(err.Error(), "This combination of ServicePlan and Organization is already taken") {
			log.Printf(
				"[INFO] skipping creation of service access %s on org %s with plan %s",
				client.Config().ApiEndpoint,
				serviceAccess.Service,
				serviceAccess.OrgId,
				serviceAccess.Plan,
			)
			return nil
		}
		return err
	}
	return nil
}
func (c CfServiceBrokerResource) getService(serviceBroker models.ServiceBroker, serviceName string) models.ServiceOffering {
	for _, service := range serviceBroker.Services {
		if service.Label == serviceName || service.GUID == serviceName {
			return service
		}
	}
	return models.ServiceOffering{}
}
func (c CfServiceBrokerResource) getServicePlan(service models.ServiceOffering, planName string) models.ServicePlanFields {
	for _, plan := range service.Plans {
		if (plan.Name == planName || plan.GUID == planName) && plan.ServiceOfferingGUID == service.GUID {
			return plan
		}
	}
	return models.ServicePlanFields{}
}
func (c CfServiceBrokerResource) getServiceBrokerFromCf(client cf_client.Client, guid string) (models.ServiceBroker, error) {
	serviceBroker, err := client.ServiceBrokers().FindByGUID(guid)
	if err != nil {
		if strings.Contains(err.Error(), "status code: 404") {
			return models.ServiceBroker{}, nil
		}
		return models.ServiceBroker{}, err
	}
	services, err := client.Services().ListServicesFromBroker(guid)
	if err != nil {
		return models.ServiceBroker{}, err
	}
	for i, service := range services {
		servicePlans, err := client.ServicePlans().ListPlansFromManyServices([]string{service.GUID})
		if err != nil {
			return models.ServiceBroker{}, err
		}
		services[i].Plans = servicePlans
	}
	serviceBroker.Services = services
	return serviceBroker, nil
}
func (c CfServiceBrokerResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	name := d.Get("name").(string)
	serviceBroker, err := client.ServiceBrokers().FindByName(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	d.SetId(serviceBroker.GUID)
	return true, nil
}
func (c CfServiceBrokerResource) diffServicesAccess(client cf_client.Client, servicesAccessSrc,
servicesAccessDest []ServiceAccess) (toDelete []ServiceAccess, toCreate []ServiceAccess) {

	toDelete = make([]ServiceAccess, 0)
	toCreate = make([]ServiceAccess, 0)

	for _, serviceAccessSrc := range servicesAccessSrc {
		if !c.containsServiceAccess(servicesAccessDest, serviceAccessSrc) {
			toDelete = append(toDelete, serviceAccessSrc)
		}
	}
	for _, serviceAccessDest := range servicesAccessDest {
		if !c.containsServiceAccess(servicesAccessSrc, serviceAccessDest) {
			toCreate = append(toCreate, serviceAccessDest)
		}
	}
	return
}
func (c CfServiceBrokerResource) containsServiceAccess(servicesAccess []ServiceAccess, serviceAccessToFind ServiceAccess) bool {
	for _, serviceAccess := range servicesAccess {
		if serviceAccess.OrgId == serviceAccessToFind.OrgId &&
			serviceAccess.Plan == serviceAccessToFind.Plan &&
			serviceAccess.Service == serviceAccessToFind.Service {
			return true
		}
	}
	return false
}
func (c CfServiceBrokerResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	brokerName := d.Get("name").(string)
	broker, err := c.getServiceBrokerFromCf(client, d.Id())
	if err != nil {
		return err
	}
	if broker.GUID == "" {
		log.Printf(
			"[WARN] removing service broker %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			brokerName,
		)
		d.SetId("")
		return nil
	}
	d.Set("name", broker.Name)
	d.Set("url", broker.URL)
	d.Set("username", broker.Username)

	servicesAccess, err := c.retrieveServicesAccessFromBroker(client, broker)

	serviceAccessSchema := schema.NewSet(d.Get("service_access").(*schema.Set).F, make([]interface{}, 0))
	for _, serviceAccess := range servicesAccess {
		serviceAccessSchema.Add(c.transformServicesAccessToMap(serviceAccess))
	}
	d.Set("service_access", serviceAccessSchema)
	return nil
}
func (c CfServiceBrokerResource) Update(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	brokerName := d.Get("name").(string)
	broker, err := c.resourceObject(d, meta)
	if err != nil {
		return err
	}
	brokerCf, err := c.getServiceBrokerFromCf(client, d.Id())
	if err != nil {
		return err
	}
	if brokerCf.GUID == "" {
		log.Printf(
			"[WARN] removing service broker %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			brokerName,
		)
		d.SetId("")
		return nil
	}
	if broker.Name != brokerCf.Name ||
		broker.URL != brokerCf.URL ||
		broker.Username != brokerCf.Username {
		err = client.ServiceBrokers().Update(broker)
		if err != nil {
			return err
		}
	}
	servicesAccessDest, err := c.getFullServicesAccessDef(client, brokerCf, c.serviceAccessObjects(d))
	if err != nil {
		return err
	}
	servicesAccessSrc, err := c.retrieveServicesAccessFromBroker(client, brokerCf)
	if err != nil {
		return err
	}
	servicesAccessSrc, err = c.getFullServicesAccessDef(client, brokerCf, servicesAccessSrc)

	toDelete, toCreate := c.diffServicesAccess(client, servicesAccessSrc, servicesAccessDest)

	if len(toDelete) == 0 && len(toCreate) == 0 {
		return nil
	}
	for _, serviceAccess := range toCreate {
		service := c.getService(brokerCf, serviceAccess.Service)
		if service.GUID == "" {
			return errors.New(fmt.Sprintf("Service '%s' doesn't exist in broker '%s'.",
				serviceAccess.Service, broker.Name))
		}
		err := c.updateServiceAccessWithPlanAndOrg(client, service, serviceAccess)
		if err != nil {
			return err
		}
	}
	for _, serviceAccess := range toDelete {
		service := c.getService(brokerCf, serviceAccess.Service)
		if service.GUID == "" {
			return errors.New(fmt.Sprintf("Service '%s' doesn't exist in broker '%s'.",
				serviceAccess.Service, broker.Name))
		}
		plan := c.getServicePlan(service, serviceAccess.Plan)
		if plan.GUID == "" {
			return errors.New(fmt.Sprintf("Plan '%s' doesn't exist in service '%s'.",
				serviceAccess.Service, serviceAccess.Plan))
		}
		planVisibility, err := c.getPlanVisibilityForPlanAndOrg(client, plan.GUID, serviceAccess.OrgId)
		if err != nil {
			return err
		}
		if planVisibility.GUID == "" {
			continue
		}
		err = client.ServicePlanVisibilities().Delete(planVisibility.GUID)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c CfServiceBrokerResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	return client.ServiceBrokers().Delete(d.Id())
}
func (c CfServiceBrokerResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"url": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ValidateFunc: func(elem interface{}, index string) ([]string, []error) {
				url := elem.(string)
				if IsWebURL(url) {
					return make([]string, 0), make([]error, 0)
				}
				errMsg := fmt.Sprintf(
					"Url '%s' is not a valid url. It must begin with http:// or https://",
					url,
				)
				err := errors.New(errMsg)
				return make([]string, 0), []error{err}
			},
		},
		"username": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"password": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"service_access": &schema.Schema{
			Type:     schema.TypeSet,
			Required: true,

			Elem:     &schema.Resource{
				Schema: map[string]*schema.Schema{
					"service": &schema.Schema{
						Type:     schema.TypeString,
						Required: true,
					},
					"plan": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
					},
					"org_id": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
					},
				},
			},
			Set: func(v interface{}) int {
				var buf bytes.Buffer
				m := v.(map[string]interface{})
				buf.WriteString(fmt.Sprintf("%s-", m["service"].(string)))
				buf.WriteString(fmt.Sprintf("%s-", m["plan"].(string)))
				buf.WriteString(fmt.Sprintf("%s-", m["org_id"].(string)))
				return hashcode.String(buf.String())
			},
		},
	}
}