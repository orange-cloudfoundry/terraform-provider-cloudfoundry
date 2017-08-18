package resources

import (
	"bytes"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/formatters"
	"code.cloudfoundry.org/cli/cf/models"
	"fmt"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/bitsmanager"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/common"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/rewind"
	"github.com/viant/toolbox"
	"log"
	"strings"
	"time"
)

const (
	stateStopped = "STOPPED"
	stateStarted = "STARTED"
)

type CfAppsResource struct {
	CfResource
}
type AppParams struct {
	models.AppParams
	RouteIds   []string
	ServiceIds []string
	Path       string
}

func (c CfAppsResource) resourceObject(d *schema.ResourceData) (AppParams, error) {
	state := stateStopped
	buildpack := d.Get("buildpack").(string)
	name := d.Get("name").(string)
	spaceGUID := d.Get("space_id").(string)
	instances := d.Get("instances").(int)
	memory, err := formatters.ToMegabytes(d.Get("memory").(string))
	if err != nil {
		return AppParams{}, err
	}
	diskQuota, err := formatters.ToMegabytes(d.Get("disk_quota").(string))
	if err != nil {
		return AppParams{}, err
	}
	stackGUID := d.Get("stack_id").(string)
	command := d.Get("command").(string)
	healthCheckType := d.Get("health_check_type").(string)
	healthCheckTimeout := d.Get("health_check_timeout").(int)
	healthCheckHTTPEndpoint := d.Get("health_check_http_endpoint").(string)
	dockerImage := d.Get("docker_image").(string)
	diego := d.Get("diego").(bool)
	enableSSH := d.Get("enable_ssh").(bool)
	ports := common.SchemaSetToIntList(d.Get("ports").(*schema.Set))
	if diego && len(ports) == 0 {
		ports = append(ports, 8080)
	}
	routeIds := common.SchemaSetToStringList(d.Get("routes").(*schema.Set))
	serviceIds := common.SchemaSetToStringList(d.Get("services").(*schema.Set))
	envVarSchema := d.Get("env_var").(*schema.Set)
	envVars := make(map[string]interface{})
	for _, elm := range envVarSchema.List() {
		envVar := elm.(map[string]interface{})
		envVars[envVar["key"].(string)] = envVar["value"].(string)
	}
	return AppParams{
		AppParams: models.AppParams{
			BuildpackURL:            &buildpack,
			Name:                    &name,
			SpaceGUID:               &spaceGUID,
			InstanceCount:           &instances,
			Memory:                  &memory,
			DiskQuota:               &diskQuota,
			StackGUID:               &stackGUID,
			Command:                 common.VarToStrPointer(command),
			HealthCheckType:         &healthCheckType,
			HealthCheckHTTPEndpoint: &healthCheckHTTPEndpoint,
			HealthCheckTimeout:      common.VarToIntPointer(healthCheckTimeout),
			DockerImage:             common.VarToStrPointer(dockerImage),
			Diego:                   &diego,
			EnableSSH:               &enableSSH,
			AppPorts:                &ports,
			State:                   &state,
			EnvironmentVars:         &envVars,
		},
		RouteIds:   routeIds,
		ServiceIds: serviceIds,
	}, nil
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
	if ok, _ := c.Exists(d, meta); ok {
		log.Printf(
			"[INFO] updating app %s/%s instead of creating it because it already exists on your Cloud Foundry",
			client.Config().ApiEndpoint,
			d.Get("name").(string),
		)
	}
	return c.createOrUpdate(d, meta)
}
func (c CfAppsResource) createOrUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	if d.Id() == "" {
		return c.createApp(d, meta, d.Get("started").(bool), true)
	}
	if c.IsRoutesUpdate(d) {
		app, err := client.Finder().GetAppFromCf(d.Id())
		if err != nil {
			return err
		}
		return c.updateRoutes(d, meta, app)
	}
	if c.IsScaleUpdate(d) {
		instances := d.Get("instances").(int)
		_, err := client.Applications().Update(d.Id(), models.AppParams{InstanceCount: &instances})
		return err
	}
	if c.IsRenameUpdate(d) {
		name := d.Get("name").(string)
		_, err := client.Applications().Update(d.Id(), models.AppParams{Name: &name})
		return err
	}
	if c.IsScaleAndRename(d) {
		name := d.Get("name").(string)
		instances := d.Get("instances").(int)
		_, err := client.Applications().Update(d.Id(), models.AppParams{Name: &name, InstanceCount: &instances})
		return err
	}
	if c.IsBitsDiff(d) && d.Get("no_blue_green_deploy").(bool) {
		a := models.Application{}
		a.GUID = d.Id()
		err := c.stopApp(client, a)
		if err != nil {
			return err
		}
		err = c.SendBits(d, meta)
		if err != nil {
			return err
		}
		return c.startApp(client, a)
	}
	if c.IsBitsDiff(d) {
		return c.updateBgDeploy(d, meta)
	}
	if d.Get("no_blue_green_restage").(bool) {
		appParams, err := c.resourceObject(d)
		if err != nil {
			return err
		}
		a := models.Application{}
		a.GUID = d.Id()
		_, err = client.Applications().Update(d.Id(), appParams.AppParams)
		if err != nil {
			return err
		}
		return c.restartApp(client, a)
	}
	return c.updateBgRestage(d, meta)
}
func (c CfAppsResource) updateBg(actionList []rewind.Action) error {
	actions := rewind.Actions{
		Actions:              actionList,
		RewindFailureMessage: "Oh no. Something's gone wrong. I've tried to roll back but you should check to see if everything is OK.",
	}
	return actions.Execute()
}
func (c CfAppsResource) updateBgRestage(d *schema.ResourceData, meta interface{}) error {
	err := c.updateBg(c.rewindActionsBgRestage(d, meta))
	if err != nil {
		return fmt.Errorf("Error when trying to update the app %s in blue-green restage mode: %s", d.Get("name").(string), err.Error())
	}
	return nil
}
func (c CfAppsResource) updateBgDeploy(d *schema.ResourceData, meta interface{}) error {
	err := c.updateBg(c.rewindActionsBgDeploy(d, meta))
	if err != nil {
		return fmt.Errorf("Error when trying to update the app %s in blue-green deploy mode: %s", d.Get("name").(string), err.Error())
	}
	return nil
}
func (c CfAppsResource) updateRoutes(d *schema.ResourceData, meta interface{}, a models.Application) error {
	client := meta.(cf_client.Client)
	currentRoutes := make([]string, 0)
	if d.HasChange("routes") {
		currentTfRoutes, _ := d.GetChange("routes")
		currentRoutes = common.SchemaSetToStringList(currentTfRoutes.(*schema.Set))
	}
	return c.BindRoutes(client, a, common.SchemaSetToStringList(d.Get("routes").(*schema.Set)), currentRoutes)
}
func (c CfAppsResource) createApp(d *schema.ResourceData, meta interface{}, started bool, sendBits bool) error {
	client := meta.(cf_client.Client)
	appParams, err := c.resourceObject(d)
	if err != nil {
		return err
	}
	app, err := client.Applications().Create(appParams.AppParams)
	if err != nil {
		return err
	}
	d.SetId(app.GUID)
	err = c.updateRoutes(d, meta, app)
	if err != nil {
		return err
	}
	currentServices := make([]string, 0)
	if d.HasChange("services") {
		currentTfServices, _ := d.GetChange("services")
		currentServices = common.SchemaSetToStringList(currentTfServices.(*schema.Set))
	}
	err = c.BindServices(client, app, appParams.ServiceIds, currentServices)
	if err != nil {
		return err
	}

	if sendBits {
		err = c.SendBits(d, meta)
		if err != nil {
			return err
		}
	}
	if !started {
		return nil
	}
	err = c.startApp(client, app)
	if err != nil {
		return err
	}
	return nil
}
func (c CfAppsResource) stopApp(client cf_client.Client, a models.Application) error {
	state := stateStopped
	_, err := client.Applications().Update(a.GUID, models.AppParams{State: &state})
	if err != nil {
		return err
	}
	return nil
}
func (c CfAppsResource) rewindActionsBgDeploy(d *schema.ResourceData, meta interface{}) []rewind.Action {
	client := meta.(cf_client.Client)
	oldAppName, newAppName := d.GetChange("name")
	origAppName := oldAppName.(string)
	if origAppName == "" {
		origAppName = newAppName.(string)
	}
	origAppGuid := d.Id()
	return []rewind.Action{
		{
			Forward: func() error {
				return c.renameApplication(client, origAppGuid, venerableAppName(origAppName))
			},
		},
		{
			Forward: func() error {
				return c.createApp(d, meta, d.Get("started").(bool), true)
			},
			ReversePrevious: func() error {
				client.Applications().Delete(d.Id())
				return c.renameApplication(client, origAppGuid, origAppName)
			},
		},
		{
			Forward: func() error {
				return client.Applications().Delete(origAppGuid)
			},
		},
	}
}
func (c CfAppsResource) rewindActionsBgRestage(d *schema.ResourceData, meta interface{}) []rewind.Action {
	client := meta.(cf_client.Client)
	bm := c.MakeBitsManager(meta)
	oldAppName, newAppName := d.GetChange("name")
	origAppName := oldAppName.(string)
	if origAppName == "" {
		origAppName = newAppName.(string)
	}
	origAppGuid := d.Id()
	defaultReverse := func() error {
		client.Applications().Delete(d.Id())
		return c.renameApplication(client, origAppGuid, origAppName)
	}
	return []rewind.Action{
		{
			Forward: func() error {
				return c.renameApplication(client, origAppGuid, venerableAppName(origAppName))
			},
		},
		{
			Forward: func() error {
				return c.createApp(d, meta, false, false)
			},
			ReversePrevious: defaultReverse,
		},
		{
			Forward: func() error {
				return bm.CopyBits(origAppGuid, d.Id())
			},
			ReversePrevious: defaultReverse,
		},
		{
			Forward: func() error {
				if !d.Get("started").(bool) {
					return nil
				}
				return c.startApp(client, models.Application{ApplicationFields: models.ApplicationFields{GUID: d.Id()}})
			},
			ReversePrevious: defaultReverse,
		},
		{
			Forward: func() error {
				return client.Applications().Delete(origAppGuid)
			},
		},
	}
}
func venerableAppName(appName string) string {
	return fmt.Sprintf("%s-venerable", appName)
}
func (c CfAppsResource) renameApplication(client cf_client.Client, appGuid string, newName string) error {
	finalName := newName
	_, err := client.Applications().Update(appGuid, models.AppParams{Name: &finalName})
	if err != nil {
		return err
	}
	return nil
}
func (c CfAppsResource) restartApp(client cf_client.Client, a models.Application) error {
	err := c.stopApp(client, a)
	if err != nil {
		return err
	}
	err = c.startApp(client, a)
	if err != nil {
		return err
	}
	return nil
}
func (c CfAppsResource) IsRoutesUpdate(d *schema.ResourceData) bool {
	return c.IsKeyUpdate(d, "routes")
}
func (c CfAppsResource) IsKeyUpdate(d *schema.ResourceData, key string) bool {
	if !d.HasChange(key) {
		return false
	}
	for schemaKey, _ := range c.Schema() {
		if d.HasChange(schemaKey) && schemaKey != key {
			return false
		}
	}
	return true
}
func (c CfAppsResource) IsScaleAndRename(d *schema.ResourceData) bool {
	if !d.HasChange("instances") || !d.HasChange("name") {
		return false
	}
	for schemaKey, _ := range c.Schema() {
		if d.HasChange(schemaKey) && schemaKey != "instances" && schemaKey != "name" {
			return false
		}
	}
	return true
}
func (c CfAppsResource) IsRenameUpdate(d *schema.ResourceData) bool {
	return c.IsKeyUpdate(d, "name")
}
func (c CfAppsResource) IsScaleUpdate(d *schema.ResourceData) bool {
	return c.IsKeyUpdate(d, "instances")
}
func (c CfAppsResource) startApp(client cf_client.Client, a models.Application) error {
	state := stateStarted
	_, err := client.Applications().Update(a.GUID, models.AppParams{State: &state})
	if err != nil {
		return err
	}
	err = common.PollingWithTimeout(func() (bool, error) {
		app, err := client.Applications().GetApp(a.GUID)
		if err != nil {
			return true, err
		}
		if app.PackageState == "STAGED" {
			return true, nil
		}
		if app.PackageState == "FAILED" {
			return true, fmt.Errorf("Staging failed for app %s", a.Name)
		}
		return false, nil
	}, 5*time.Second, 15*time.Minute)
	if err != nil {
		return c.createErrorFromLog(err, client, a)
	}
	err = common.Polling(func() (bool, error) {
		appInstances, err := client.AppInstances().GetInstances(a.GUID)
		if err != nil {
			return true, err
		}
		if a.InstanceCount == 0 {
			return true, nil
		}
		for i, instance := range appInstances {
			if instance.State == models.InstanceStarting {
				continue
			}
			if instance.State == models.InstanceRunning {
				return true, nil
			}
			return true, fmt.Errorf("Instance %d failed with state %s for app %s", i, instance.State, a.Name)
		}

		return false, nil
	}, 5*time.Second)
	if err != nil {
		return c.createErrorFromLog(err, client, a)
	}
	return nil
}
func (c CfAppsResource) createErrorFromLog(parentErr error, client cf_client.Client, a models.Application) error {
	loggables, logErr := client.Logs().RecentLogsFor(a.GUID)
	if logErr != nil {
		return fmt.Errorf("%s and failed to retrieve logs (error: %s)", parentErr.Error(), logErr.Error())
	}
	logs := ""
	for _, loggable := range loggables {
		logs += "\n\t" + loggable.ToSimpleLog()
	}
	return fmt.Errorf("%s:%s", parentErr.Error(), logs)
}
func (c CfAppsResource) BindServices(client cf_client.Client, a models.Application, newServices, currentServices []string) error {
	if len(newServices) == 0 {
		return nil
	}
	currentBindings, err := client.Finder().GetServiceBindingsFromApp(a.GUID)
	if err != nil {
		return err
	}
	var toAdd = make([]string, 0)
	toolbox.FilterSliceElements(newServices, func(item string) bool {
		for _, binding := range currentBindings {
			if item == binding.GUID {
				return false
			}
		}
		return true
	}, &toAdd)
	for _, service := range toAdd {
		err := client.ServiceBinding().Create(service, a.GUID, make(map[string]interface{}))
		if err != nil {
			return err
		}
	}
	var toDelete = make([]models.ServiceBindingFields, 0)
	toolbox.FilterSliceElements(currentBindings, func(item models.ServiceBindingFields) bool {
		for _, service := range newServices {
			if item.GUID == service || !toolbox.HasSliceAnyElements(currentServices, item.GUID) {
				return false
			}
		}
		return true
	}, &toDelete)
	for _, service := range toDelete {
		_, err := client.ServiceBinding().Delete(models.ServiceInstance{
			ServiceBindings: []models.ServiceBindingFields{service},
		}, a.GUID)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c CfAppsResource) BindRoutes(client cf_client.Client, a models.Application, newRoutes, currentRoutes []string) error {
	if len(newRoutes) == 0 {
		return nil
	}
	var toAdd = make([]string, 0)
	toolbox.FilterSliceElements(newRoutes, func(item string) bool {
		for _, appRoute := range a.Routes {
			if item == appRoute.GUID {
				return false
			}
		}
		return true
	}, &toAdd)
	for _, route := range toAdd {
		err := client.Route().Bind(route, a.GUID)
		if err != nil {
			return err
		}
	}
	var toDelete = make([]models.RouteSummary, 0)
	toolbox.FilterSliceElements(a.Routes, func(item models.RouteSummary) bool {
		for _, route := range newRoutes {
			if item.GUID == route || !toolbox.HasSliceAnyElements(currentRoutes, item.GUID) {
				return false
			}
		}
		return true
	}, &toDelete)
	for _, appRoute := range toDelete {
		err := client.Route().Unbind(appRoute.GUID, a.GUID)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c CfAppsResource) SendBits(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	bm := c.MakeBitsManager(meta)
	err := bm.Upload(d.Id(), d.Get("path").(string))
	if err != nil {
		return err
	}
	localSha1, err := bm.GetSha1(d.Get("path").(string))
	if err != nil {
		return err
	}
	d.Set("path_sha1", localSha1)
	rmtSha1, err := client.ApplicationBits().GetApplicationSha1(d.Id())
	if err != nil {
		return err
	}
	d.Set("remote_sha1", rmtSha1)
	return nil
}
func (c CfAppsResource) IsBitsDiff(d *schema.ResourceData) bool {
	return d.HasChange("bits_has_changed")
}
func (c CfAppsResource) updateBitsDiff(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	bm := c.MakeBitsManager(meta)
	isDiffLocal, sha1Local, err := bm.IsDiff(d.Get("path").(string), d.Get("path_sha1").(string))
	if err != nil {
		return err
	}
	isDiffRmt, sha1Rmt, err := client.ApplicationBits().IsDiff(d.Id(), d.Get("remote_sha1").(string))
	if err != nil {
		return err
	}
	if isDiffLocal || isDiffRmt {
		d.Set("path_sha1", sha1Local)
		d.Set("remote_sha1", sha1Rmt)
		d.Set("bits_has_changed", "modified")
		return nil
	}
	d.Set("bits_has_changed", "")
	return nil
}
func (c CfAppsResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)

	app, err := client.Finder().GetAppFromCf(d.Id())
	if err != nil {
		return err
	}
	currentBindings, err := client.Finder().GetServiceBindingsFromApp(d.Id())
	if err != nil {
		return err
	}
	if strings.ToUpper(app.State) == stateStarted {
		d.Set("started", true)
	} else {
		d.Set("started", false)
	}
	d.Set("buildpack", app.Buildpack)
	d.Set("name", app.Name)
	d.Set("space_id", app.SpaceGUID)
	d.Set("instances", app.InstanceCount)
	d.Set("memory", formatters.ByteSize(app.Memory*formatters.MEGABYTE))
	d.Set("disk_quota", formatters.ByteSize(app.DiskQuota*formatters.MEGABYTE))
	d.Set("stack_id", app.Stack.GUID)
	d.Set("command", app.Command)
	d.Set("health_check_type", app.HealthCheckType)
	d.Set("health_check_timeout", app.HealthCheckTimeout)
	d.Set("health_check_http_endpoint", app.HealthCheckHTTPEndpoint)
	d.Set("docker_image", app.DockerImage)
	d.Set("diego", app.Diego)
	d.Set("enable_ssh", app.EnableSSH)
	schemaEnvVar := schema.NewSet(d.Get("env_var").(*schema.Set).F, make([]interface{}, 0))
	for key, value := range app.EnvironmentVars {
		m := map[string]interface{}{
			"key":   key,
			"value": value,
		}
		schemaEnvVar.Add(m)
	}
	d.Set("env_var", schemaEnvVar)
	currentRoutes := common.SchemaSetToStringList(d.Get("routes").(*schema.Set))
	schemaRoutes := schema.NewSet(d.Get("routes").(*schema.Set).F, make([]interface{}, 0))
	for _, route := range app.Routes {
		if !toolbox.HasSliceAnyElements(currentRoutes, route.GUID) {
			continue
		}
		schemaRoutes.Add(route.GUID)
	}
	d.Set("routes", schemaRoutes)

	currentServices := common.SchemaSetToStringList(d.Get("services").(*schema.Set))
	schemaServices := schema.NewSet(d.Get("services").(*schema.Set).F, make([]interface{}, 0))
	for _, binding := range currentBindings {
		if !toolbox.HasSliceAnyElements(currentServices, binding.ServiceInstanceGUID) {
			continue
		}
		schemaServices.Add(binding.ServiceInstanceGUID)
	}
	d.Set("services", schemaServices)
	return c.updateBitsDiff(d, meta)
}
func (c CfAppsResource) Update(d *schema.ResourceData, meta interface{}) error {
	return c.createOrUpdate(d, meta)
}
func (c CfAppsResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	return client.Applications().Delete(d.Id())
}
func (c CfAppsResource) existsWithoutSpaceId(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	orgs, err := client.Organizations().ListOrgs(-1)
	if err != nil {
		return false, err
	}

	for _, org := range orgs {
		err = client.Spaces().ListSpacesFromOrg(org.GUID, func(space models.Space) bool {
			for _, app := range space.Applications {
				if app.Name == d.Get("name").(string) {
					d.SetId(app.GUID)
					return false
				}
			}
			return true
		})
		if err != nil {
			return false, err
		}
	}
	return d.Id() != "", nil
}
func (c CfAppsResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)

	if d.Get("space_id").(string) == "" {
		return c.existsWithoutSpaceId(d, meta)
	}
	if d.Id() != "" {
		app, err := client.Finder().GetAppFromCf(d.Id())
		if err != nil {
			return false, err
		}
		return app.GUID != "", nil
	}
	app, err := client.Applications().ReadFromSpace(d.Get("name").(string), d.Get("space_id").(string))
	if err != nil {
		if _, ok := err.(*errors.ModelNotFoundError); ok {
			return false, nil
		}
		return false, err
	}
	d.SetId(app.GUID)
	return true, nil
}
func (c CfAppsResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"started": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
		},
		"space_id": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"instances": &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
			Default:  1,
		},
		"memory": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Default:  "512M",
		},
		"disk_quota": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Default:  "1G",
		},
		"stack_id": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"command": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"buildpack": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"health_check_type": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Default:  "port",
		},
		"health_check_http_endpoint": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"health_check_timeout": &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
		},
		"docker_image": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
		},
		"diego": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
		},
		"enable_ssh": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		"ports": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeInt},
			Set: func(v interface{}) int {
				return v.(int)
			},
		},
		"routes": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Set:      schema.HashString,
		},
		"services": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Set:      schema.HashString,
		},
		"env_var": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"key": &schema.Schema{
						Type:     schema.TypeString,
						Required: true,
					},
					"value": &schema.Schema{
						Type:     schema.TypeString,
						Required: true,
					},
				},
			},
			Set: func(v interface{}) int {
				var buf bytes.Buffer
				m := v.(map[string]interface{})
				buf.WriteString(fmt.Sprintf("%s-", m["key"].(string)))
				buf.WriteString(fmt.Sprintf("%s-", m["value"].(string)))
				return hashcode.String(buf.String())
			},
		},
		"no_blue_green_restage": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		"no_blue_green_deploy": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
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
func (c CfAppsResource) DataSourceSchema() map[string]*schema.Schema {
	return CreateDataSourceSchema(c)
}
func (c CfAppsResource) DataSourceRead(d *schema.ResourceData, meta interface{}) error {
	return CreateDataSourceReadFunc(c)(d, meta)
}
