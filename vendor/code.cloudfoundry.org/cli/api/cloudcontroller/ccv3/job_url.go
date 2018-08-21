package ccv3

import (
	"bytes"

	"code.cloudfoundry.org/cli/api/cloudcontroller"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/internal"
)

// JobURL is the URL to a given Job.
type JobURL string

// DeleteApplication deletes the app with the given app GUID. Returns back a
// resulting job URL to poll.
func (client *Client) DeleteApplication(appGUID string) (JobURL, Warnings, error) {
	request, err := client.newHTTPRequest(requestOptions{
		RequestName: internal.DeleteApplicationRequest,
		URIParams:   internal.Params{"app_guid": appGUID},
	})
	if err != nil {
		return "", nil, err
	}

	response := cloudcontroller.Response{}
	err = client.connection.Make(request, &response)

	return JobURL(response.ResourceLocationURL), response.Warnings, err
}

// UpdateApplicationApplyManifest applies the manifest to the given
// application. Returns back a resulting job URL to poll.
func (client *Client) UpdateApplicationApplyManifest(appGUID string, rawManifest []byte) (JobURL, Warnings, error) {
	request, err := client.newHTTPRequest(requestOptions{
		RequestName: internal.PostApplicationActionApplyManifest,
		URIParams:   map[string]string{"app_guid": appGUID},
		Body:        bytes.NewReader(rawManifest),
	})

	if err != nil {
		return "", nil, err
	}

	request.Header.Set("Content-Type", "application/x-yaml")

	response := cloudcontroller.Response{}
	err = client.connection.Make(request, &response)

	return JobURL(response.ResourceLocationURL), response.Warnings, err
}
