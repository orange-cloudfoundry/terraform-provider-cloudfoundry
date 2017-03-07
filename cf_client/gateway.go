package cf_client

import (
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/net"
	"code.cloudfoundry.org/cli/cf/terminal"
	"code.cloudfoundry.org/cli/cf/trace"
	"io/ioutil"
	"time"
)

type CloudFoundryGateways struct {
	CloudControllerGateway net.Gateway
	UAAGateway             net.Gateway
	Config                 coreconfig.ReadWriter
}

func NewCloudControllerGateway(config coreconfig.ReadWriter, logger trace.Printer) net.Gateway {
	return net.NewCloudControllerGateway(config, time.Now, createUi(logger), logger, "5")
}
func createUi(logger trace.Printer) terminal.UI {
	return terminal.NewUI(ioutil.NopCloser(nil), ioutil.Discard, terminal.NewTeePrinter(ioutil.Discard), logger)
}
func NewUAAGateway(config coreconfig.ReadWriter, logger trace.Printer) net.Gateway {
	return net.NewUAAGateway(config, createUi(logger), logger, "5")
}

func NewCloudFoundryGateways(config coreconfig.ReadWriter, logger trace.Printer) CloudFoundryGateways {
	return CloudFoundryGateways{
		CloudControllerGateway: NewCloudControllerGateway(config, logger),
		UAAGateway:             NewUAAGateway(config, logger),
		Config:                 config,
	}
}
