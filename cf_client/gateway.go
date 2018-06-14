package cf_client

import (
	"code.cloudfoundry.org/cli/api/uaa"
	"code.cloudfoundry.org/cli/api/uaa/noaabridge"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/net"
	"code.cloudfoundry.org/cli/cf/terminal"
	"code.cloudfoundry.org/cli/cf/trace"
	"crypto/tls"
	"github.com/cloudfoundry/noaa/consumer"
	"io/ioutil"
	"net/http"
	"time"
)

type CloudFoundryGateways struct {
	CloudControllerGateway net.Gateway
	UAAGateway             net.Gateway
	Config                 coreconfig.ReadWriter
}

func NewCloudControllerGateway(config coreconfig.ReadWriter, logger trace.Printer, uaaClient *uaa.Client) net.Gateway {
	gw := net.NewCloudControllerGateway(config, time.Now, createUi(logger), logger, "5")
	gw.SetTokenRefresher(noaabridge.NewTokenRefresher(uaaClient, config))
	return gw
}
func createUi(logger trace.Printer) terminal.UI {
	return terminal.NewUI(ioutil.NopCloser(nil), ioutil.Discard, terminal.NewTeePrinter(ioutil.Discard), logger)
}
func NewUAAGateway(config coreconfig.ReadWriter, logger trace.Printer, uaaClient *uaa.Client) net.Gateway {
	gw := net.NewUAAGateway(config, createUi(logger), logger, "5")
	gw.SetTokenRefresher(noaabridge.NewTokenRefresher(uaaClient, config))
	return gw
}
func NewNOAAClient(config coreconfig.ReadWriter, uaaClient *uaa.Client) *consumer.Consumer {
	client := consumer.New(
		config.DopplerEndpoint(),
		&tls.Config{
			InsecureSkipVerify: config.IsSSLDisabled(),
		},
		http.ProxyFromEnvironment,
	)
	client.RefreshTokenFrom(noaabridge.NewTokenRefresher(uaaClient, config))
	return client
}
func NewCloudFoundryGateways(config coreconfig.ReadWriter, logger trace.Printer, uaaClient *uaa.Client) CloudFoundryGateways {
	return CloudFoundryGateways{
		CloudControllerGateway: NewCloudControllerGateway(config, logger, uaaClient),
		UAAGateway:             NewUAAGateway(config, logger, uaaClient),
		Config:                 config,
	}
}
