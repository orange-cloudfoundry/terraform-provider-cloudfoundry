package v3

import (
	"net/http"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/pushaction"
	"code.cloudfoundry.org/cli/actor/sharedaction"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/actor/v3action"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccversion"
	"code.cloudfoundry.org/cli/command"
	"code.cloudfoundry.org/cli/command/translatableerror"
	sharedV2 "code.cloudfoundry.org/cli/command/v2/shared"
	"code.cloudfoundry.org/cli/command/v3/shared"
)

//go:generate counterfeiter . OriginalV2PushActor

type OriginalV2PushActor interface {
	CreateAndMapDefaultApplicationRoute(orgGUID string, spaceGUID string, app v2action.Application) (pushaction.Warnings, error)
}

//go:generate counterfeiter . OriginalV3PushActor

type OriginalV3PushActor interface {
	CloudControllerAPIVersion() string
	CreateAndUploadBitsPackageByApplicationNameAndSpace(appName string, spaceGUID string, bitsPath string) (v3action.Package, v3action.Warnings, error)
	CreateDockerPackageByApplicationNameAndSpace(appName string, spaceGUID string, dockerImageCredentials v3action.DockerImageCredentials) (v3action.Package, v3action.Warnings, error)
	CreateApplicationInSpace(app v3action.Application, spaceGUID string) (v3action.Application, v3action.Warnings, error)
	GetApplicationByNameAndSpace(appName string, spaceGUID string) (v3action.Application, v3action.Warnings, error)
	GetApplicationSummaryByNameAndSpace(appName string, spaceGUID string, withObfuscatedValues bool) (v3action.ApplicationSummary, v3action.Warnings, error)
	GetStreamingLogsForApplicationByNameAndSpace(appName string, spaceGUID string, client v3action.NOAAClient) (<-chan *v3action.LogMessage, <-chan error, v3action.Warnings, error)
	PollStart(appGUID string, warnings chan<- v3action.Warnings) error
	SetApplicationDropletByApplicationNameAndSpace(appName string, spaceGUID string, dropletGUID string) (v3action.Warnings, error)
	StagePackage(packageGUID string, appName string) (<-chan v3action.Droplet, <-chan v3action.Warnings, <-chan error)
	StartApplication(appGUID string) (v3action.Application, v3action.Warnings, error)
	StopApplication(appGUID string) (v3action.Warnings, error)
	UpdateApplication(app v3action.Application) (v3action.Application, v3action.Warnings, error)
}

func (cmd *V3PushCommand) OriginalSetup(config command.Config, ui command.UI) error {
	cmd.UI = ui
	cmd.Config = config
	sharedActor := sharedaction.NewActor(config)

	ccClient, uaaClient, err := shared.NewClients(config, ui, true, "")
	if err != nil {
		if v3Err, ok := err.(ccerror.V3UnexpectedResponseError); ok && v3Err.ResponseCode == http.StatusNotFound {
			return translatableerror.MinimumAPIVersionNotMetError{MinimumVersion: ccversion.MinVersionApplicationFlowV3}
		}

		return err
	}
	v3actor := v3action.NewActor(ccClient, config, sharedActor, nil)
	cmd.OriginalActor = v3actor

	ccClientV2, uaaClientV2, err := sharedV2.NewClients(config, ui, true)
	if err != nil {
		return err
	}

	v2Actor := v2action.NewActor(ccClientV2, uaaClientV2, config)

	cmd.SharedActor = sharedActor
	cmd.OriginalV2PushActor = pushaction.NewActor(v2Actor, v3actor, sharedActor)

	v2AppActor := v2action.NewActor(ccClientV2, uaaClientV2, config)
	cmd.NOAAClient = shared.NewNOAAClient(ccClient.Info.Logging(), config, uaaClient, ui)

	cmd.AppSummaryDisplayer = shared.AppSummaryDisplayer{
		UI:         cmd.UI,
		Config:     cmd.Config,
		Actor:      cmd.OriginalActor,
		V2AppActor: v2AppActor,
		AppName:    cmd.RequiredArgs.AppName,
	}
	cmd.PackageDisplayer = shared.NewPackageDisplayer(cmd.UI, cmd.Config)

	return nil
}

func (cmd V3PushCommand) OriginalExecute(args []string) error {
	cmd.UI.DisplayWarning(command.ExperimentalWarning)

	err := cmd.validateArgs()
	if err != nil {
		return err
	}

	err = command.MinimumAPIVersionCheck(cmd.OriginalActor.CloudControllerAPIVersion(), ccversion.MinVersionApplicationFlowV3)
	if err != nil {
		return err
	}

	err = cmd.SharedActor.CheckTarget(true, true)
	if err != nil {
		return err
	}

	user, err := cmd.Config.CurrentUser()
	if err != nil {
		return err
	}

	if !verifyBuildpacks(cmd.Buildpacks) {
		return translatableerror.ConflictingBuildpacksError{}
	}

	var app v3action.Application
	app, err = cmd.getApplication()
	if _, ok := err.(actionerror.ApplicationNotFoundError); ok {
		app, err = cmd.createApplication(user.Name)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		app, err = cmd.updateApplication(user.Name, app.GUID)
		if err != nil {
			return err
		}
	}

	pkg, err := cmd.createPackage()
	if err != nil {
		return err
	}

	if app.Started() {
		err = cmd.stopApplication(app.GUID, user.Name)
		if err != nil {
			return err
		}
	}

	if cmd.NoStart {
		return nil
	}

	dropletGUID, err := cmd.stagePackage(pkg, user.Name)
	if err != nil {
		return err
	}

	err = cmd.setApplicationDroplet(dropletGUID, user.Name)
	if err != nil {
		return err
	}

	if !cmd.NoRoute {
		err = cmd.createAndMapRoutes(app)
		if err != nil {
			return err
		}
	}

	err = cmd.startApplication(app.GUID, user.Name)
	if err != nil {
		return err
	}

	cmd.UI.DisplayText("Waiting for app to start...")

	warnings := make(chan v3action.Warnings)
	done := make(chan bool)
	go func() {
		for {
			select {
			case message := <-warnings:
				cmd.UI.DisplayWarnings(message)
			case <-done:
				return
			}
		}
	}()

	err = cmd.OriginalActor.PollStart(app.GUID, warnings)
	done <- true

	if err != nil {
		if _, ok := err.(actionerror.StartupTimeoutError); ok {
			return translatableerror.StartupTimeoutError{
				AppName:    cmd.RequiredArgs.AppName,
				BinaryName: cmd.Config.BinaryName(),
			}
		}

		return err
	}

	cmd.UI.DisplayTextWithFlavor("Showing health and status for app {{.AppName}} in org {{.OrgName}} / space {{.SpaceName}} as {{.Username}}...", map[string]interface{}{
		"AppName":   cmd.RequiredArgs.AppName,
		"OrgName":   cmd.Config.TargetedOrganization().Name,
		"SpaceName": cmd.Config.TargetedSpace().Name,
		"Username":  user.Name,
	})
	cmd.UI.DisplayNewline()

	return cmd.AppSummaryDisplayer.DisplayAppInfo()
}

func (cmd V3PushCommand) validateArgs() error {
	switch {
	case cmd.DockerImage.Path != "" && cmd.AppPath != "":
		return translatableerror.ArgumentCombinationError{
			Args: []string{"--docker-image", "-o", "-p"},
		}
	case cmd.DockerImage.Path != "" && len(cmd.Buildpacks) > 0:
		return translatableerror.ArgumentCombinationError{
			Args: []string{"-b", "--docker-image", "-o"},
		}
	case cmd.DockerUsername != "" && cmd.DockerImage.Path == "":
		return translatableerror.RequiredFlagsError{
			Arg1: "--docker-image, -o", Arg2: "--docker-username",
		}
	case cmd.DockerUsername != "" && cmd.Config.DockerPassword() == "":
		return translatableerror.DockerPasswordNotSetError{}
	}
	return nil
}

func (cmd V3PushCommand) createApplication(userName string) (v3action.Application, error) {
	appToCreate := v3action.Application{
		Name: cmd.RequiredArgs.AppName,
	}

	if cmd.DockerImage.Path != "" {
		appToCreate.LifecycleType = constant.AppLifecycleTypeDocker
	} else {
		appToCreate.LifecycleType = constant.AppLifecycleTypeBuildpack
		appToCreate.LifecycleBuildpacks = cmd.Buildpacks
	}

	app, warnings, err := cmd.OriginalActor.CreateApplicationInSpace(
		appToCreate,
		cmd.Config.TargetedSpace().GUID,
	)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return v3action.Application{}, err
	}

	cmd.UI.DisplayTextWithFlavor("Creating app {{.AppName}} in org {{.CurrentOrg}} / space {{.CurrentSpace}} as {{.CurrentUser}}...", map[string]interface{}{
		"AppName":      cmd.RequiredArgs.AppName,
		"CurrentSpace": cmd.Config.TargetedSpace().Name,
		"CurrentOrg":   cmd.Config.TargetedOrganization().Name,
		"CurrentUser":  userName,
	})

	cmd.UI.DisplayOK()
	cmd.UI.DisplayNewline()
	return app, nil
}

func (cmd V3PushCommand) getApplication() (v3action.Application, error) {
	app, warnings, err := cmd.OriginalActor.GetApplicationByNameAndSpace(cmd.RequiredArgs.AppName, cmd.Config.TargetedSpace().GUID)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return v3action.Application{}, err
	}

	return app, nil
}

func (cmd V3PushCommand) updateApplication(userName string, appGUID string) (v3action.Application, error) {
	cmd.UI.DisplayTextWithFlavor("Updating app {{.AppName}} in org {{.CurrentOrg}} / space {{.CurrentSpace}} as {{.CurrentUser}}...", map[string]interface{}{
		"AppName":      cmd.RequiredArgs.AppName,
		"CurrentSpace": cmd.Config.TargetedSpace().Name,
		"CurrentOrg":   cmd.Config.TargetedOrganization().Name,
		"CurrentUser":  userName,
	})

	appToUpdate := v3action.Application{
		GUID: appGUID,
	}

	if cmd.DockerImage.Path != "" {
		appToUpdate.LifecycleType = constant.AppLifecycleTypeDocker

	} else {
		appToUpdate.LifecycleType = constant.AppLifecycleTypeBuildpack
		appToUpdate.LifecycleBuildpacks = cmd.Buildpacks
	}

	app, warnings, err := cmd.OriginalActor.UpdateApplication(appToUpdate)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return v3action.Application{}, err
	}

	cmd.UI.DisplayOK()
	cmd.UI.DisplayNewline()
	return app, nil
}

func (cmd V3PushCommand) createAndMapRoutes(app v3action.Application) error {
	cmd.UI.DisplayText("Mapping routes...")
	routeWarnings, err := cmd.OriginalV2PushActor.CreateAndMapDefaultApplicationRoute(cmd.Config.TargetedOrganization().GUID, cmd.Config.TargetedSpace().GUID, v2action.Application{Name: app.Name, GUID: app.GUID})
	cmd.UI.DisplayWarnings(routeWarnings)
	if err != nil {
		return err
	}

	cmd.UI.DisplayOK()
	cmd.UI.DisplayNewline()
	return nil
}

func (cmd V3PushCommand) createPackage() (v3action.Package, error) {
	isDockerImage := (cmd.DockerImage.Path != "")
	err := cmd.PackageDisplayer.DisplaySetupMessage(cmd.RequiredArgs.AppName, isDockerImage)
	if err != nil {
		return v3action.Package{}, err
	}

	var (
		pkg      v3action.Package
		warnings v3action.Warnings
	)

	if isDockerImage {
		pkg, warnings, err = cmd.OriginalActor.CreateDockerPackageByApplicationNameAndSpace(cmd.RequiredArgs.AppName, cmd.Config.TargetedSpace().GUID, v3action.DockerImageCredentials{Path: cmd.DockerImage.Path, Username: cmd.DockerUsername, Password: cmd.Config.DockerPassword()})
	} else {
		pkg, warnings, err = cmd.OriginalActor.CreateAndUploadBitsPackageByApplicationNameAndSpace(cmd.RequiredArgs.AppName, cmd.Config.TargetedSpace().GUID, string(cmd.AppPath))
	}

	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return v3action.Package{}, err
	}

	cmd.UI.DisplayOK()
	cmd.UI.DisplayNewline()
	return pkg, nil
}

func (cmd V3PushCommand) stagePackage(pkg v3action.Package, userName string) (string, error) {
	cmd.UI.DisplayTextWithFlavor("Staging package for app {{.AppName}} in org {{.OrgName}} / space {{.SpaceName}} as {{.Username}}...", map[string]interface{}{
		"AppName":   cmd.RequiredArgs.AppName,
		"OrgName":   cmd.Config.TargetedOrganization().Name,
		"SpaceName": cmd.Config.TargetedSpace().Name,
		"Username":  userName,
	})

	logStream, logErrStream, logWarnings, logErr := cmd.OriginalActor.GetStreamingLogsForApplicationByNameAndSpace(cmd.RequiredArgs.AppName, cmd.Config.TargetedSpace().GUID, cmd.NOAAClient)
	cmd.UI.DisplayWarnings(logWarnings)
	if logErr != nil {
		return "", logErr
	}

	buildStream, warningsStream, errStream := cmd.OriginalActor.StagePackage(pkg.GUID, cmd.RequiredArgs.AppName)
	droplet, err := shared.PollStage(buildStream, warningsStream, errStream, logStream, logErrStream, cmd.UI)
	if err != nil {
		return "", err
	}

	cmd.UI.DisplayOK()
	cmd.UI.DisplayNewline()
	return droplet.GUID, nil
}

func (cmd V3PushCommand) setApplicationDroplet(dropletGUID string, userName string) error {
	cmd.UI.DisplayTextWithFlavor("Setting app {{.AppName}} to droplet {{.DropletGUID}} in org {{.OrgName}} / space {{.SpaceName}} as {{.Username}}...", map[string]interface{}{
		"AppName":     cmd.RequiredArgs.AppName,
		"DropletGUID": dropletGUID,
		"OrgName":     cmd.Config.TargetedOrganization().Name,
		"SpaceName":   cmd.Config.TargetedSpace().Name,
		"Username":    userName,
	})

	warnings, err := cmd.OriginalActor.SetApplicationDropletByApplicationNameAndSpace(cmd.RequiredArgs.AppName, cmd.Config.TargetedSpace().GUID, dropletGUID)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return err
	}

	cmd.UI.DisplayOK()
	cmd.UI.DisplayNewline()
	return nil
}

func (cmd V3PushCommand) startApplication(appGUID string, userName string) error {
	cmd.UI.DisplayTextWithFlavor("Starting app {{.AppName}} in org {{.OrgName}} / space {{.SpaceName}} as {{.Username}}...", map[string]interface{}{
		"AppName":   cmd.RequiredArgs.AppName,
		"OrgName":   cmd.Config.TargetedOrganization().Name,
		"SpaceName": cmd.Config.TargetedSpace().Name,
		"Username":  userName,
	})

	_, warnings, err := cmd.OriginalActor.StartApplication(appGUID)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return err
	}
	cmd.UI.DisplayOK()
	cmd.UI.DisplayNewline()
	return nil
}

func (cmd V3PushCommand) stopApplication(appGUID string, userName string) error {
	cmd.UI.DisplayTextWithFlavor("Stopping app {{.AppName}} in org {{.CurrentOrg}} / space {{.CurrentSpace}} as {{.CurrentUser}}...", map[string]interface{}{
		"AppName":      cmd.RequiredArgs.AppName,
		"CurrentSpace": cmd.Config.TargetedSpace().Name,
		"CurrentOrg":   cmd.Config.TargetedOrganization().Name,
		"CurrentUser":  userName,
	})

	warnings, err := cmd.OriginalActor.StopApplication(appGUID)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return err
	}
	cmd.UI.DisplayOK()
	cmd.UI.DisplayNewline()
	return nil
}

func verifyBuildpacks(buildpacks []string) bool {
	if len(buildpacks) < 2 {
		return true
	}

	for _, buildpack := range buildpacks {
		if buildpack == "default" || buildpack == "null" {
			return false
		}
	}
	return true
}
