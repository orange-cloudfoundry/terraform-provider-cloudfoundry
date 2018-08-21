package v2

import (
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/cli/actor/sharedaction"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/actor/v2v3action"
	"code.cloudfoundry.org/cli/actor/v3action"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccversion"
	"code.cloudfoundry.org/cli/command"
	"code.cloudfoundry.org/cli/command/flag"
	"code.cloudfoundry.org/cli/command/translatableerror"
	sharedV2 "code.cloudfoundry.org/cli/command/v2/shared"
	sharedV3 "code.cloudfoundry.org/cli/command/v3/shared"
	"code.cloudfoundry.org/cli/util/manifest"
)

//go:generate counterfeiter . CreateAppManifestActor

type CreateAppManifestActor interface {
	CreateApplicationManifestByNameAndSpace(appName string, spaceGUID string) (manifest.Application, v2v3action.Warnings, error)
	WriteApplicationManifest(manifestApp manifest.Application, manifestPath string) error
}

type CreateAppManifestCommand struct {
	RequiredArgs    flag.AppName `positional-args:"yes"`
	FilePath        flag.Path    `short:"p" description:"Specify a path for file creation. If path not specified, manifest file is created in current working directory."`
	usage           interface{}  `usage:"CF_NAME create-app-manifest APP_NAME [-p /path/to/<app-name>_manifest.yml]"`
	relatedCommands interface{}  `related_commands:"apps, push"`

	UI          command.UI
	Config      command.Config
	SharedActor command.SharedActor
	Actor       CreateAppManifestActor
}

func (cmd *CreateAppManifestCommand) Setup(config command.Config, ui command.UI) error {
	cmd.UI = ui
	cmd.Config = config
	sharedActor := sharedaction.NewActor(config)
	cmd.SharedActor = sharedActor

	ccClientV3, uaaClientV3, err := sharedV3.NewClients(config, ui, true, "")
	if err != nil {
		if v3Err, ok := err.(ccerror.V3UnexpectedResponseError); ok && v3Err.ResponseCode == http.StatusNotFound {
			return translatableerror.MinimumAPIVersionNotMetError{MinimumVersion: ccversion.MinVersionApplicationFlowV3}
		}
		return err
	}
	ccClientV2, uaaClientV2, err := sharedV2.NewClients(config, ui, true)
	if err != nil {
		return err
	}
	v2Actor := v2action.NewActor(ccClientV2, uaaClientV2, config)
	v3Actor := v3action.NewActor(ccClientV3, config, sharedActor, uaaClientV3)
	cmd.Actor = v2v3action.NewActor(v2Actor, v3Actor)

	return nil
}

func (cmd CreateAppManifestCommand) Execute(args []string) error {
	err := cmd.SharedActor.CheckTarget(true, true)
	if err != nil {
		return err
	}

	user, err := cmd.Config.CurrentUser()
	if err != nil {
		return err
	}

	cmd.UI.DisplayTextWithFlavor("Creating an app manifest from current settings of app {{.AppName}} in org {{.OrgName}} / space {{.SpaceName}} as {{.Username}}...", map[string]interface{}{
		"AppName":   cmd.RequiredArgs.AppName,
		"OrgName":   cmd.Config.TargetedOrganization().Name,
		"SpaceName": cmd.Config.TargetedSpace().Name,
		"Username":  user.Name,
	})

	manifestPath := cmd.FilePath.String()
	if manifestPath == "" {
		manifestPath = fmt.Sprintf(".%s%s_manifest.yml", string(os.PathSeparator), cmd.RequiredArgs.AppName)
	}
	manifestApp, warnings, err := cmd.Actor.CreateApplicationManifestByNameAndSpace(cmd.RequiredArgs.AppName, cmd.Config.TargetedSpace().GUID)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return err
	}
	err = cmd.Actor.WriteApplicationManifest(manifestApp, manifestPath)
	if err != nil {
		return err
	}

	cmd.UI.DisplayOK()
	cmd.UI.DisplayText("Manifest file created successfully at {{.FilePath}}", map[string]interface{}{
		"FilePath": manifestPath,
	})

	return nil
}
