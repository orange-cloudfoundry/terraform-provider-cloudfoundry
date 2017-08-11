package bitsmanager

import (
	"bytes"
	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	. "code.cloudfoundry.org/cli/cf/i18n"
	"code.cloudfoundry.org/cli/cf/net"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/gofileutils/fileutils"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"runtime"
	"time"
)

const (
	DefaultAppUploadBitsTimeout = 15 * time.Minute
)

//go:generate counterfeiter . Repository
type Job struct {
	Metadata struct {
		GUID      string    `json:"guid"`
		CreatedAt time.Time `json:"created_at"`
		URL       string    `json:"url"`
	} `json:"metadata"`
	Entity struct {
		GUID         string `json:"guid"`
		Status       string `json:"status"`
		Error        string `json:"error"`
		ErrorDetails struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			ErrorCode   string `json:"error_code"`
		} `json:"error_details"`
	} `json:"entity"`
}
type ApplicationBitsRepository interface {
	GetApplicationSha1(appGUID string) (string, error)
	IsDiff(appGUID string, currentSha1 string) (bool, string, error)
	UploadBits(appGUID string, zipFile io.ReadCloser, fileSize int64) (apiErr error)
	CopyBits(origAppGuid string, newAppGuid string) error
}

type CloudControllerApplicationBitsRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerApplicationBitsRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerApplicationBitsRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}
func (repo CloudControllerApplicationBitsRepository) IsDiff(appGUID string, currentSha1 string) (bool, string, error) {
	sha1Found, err := repo.GetApplicationSha1(appGUID)
	if err != nil {
		return true, "", err
	}
	return currentSha1 != sha1Found, sha1Found, nil
}
func (repo CloudControllerApplicationBitsRepository) GetApplicationSha1(appGUID string) (string, error) {
	// we are oblige to do the request by itself because cli is reading the full response body
	// to dump the response into a possible logger.
	// we need to read just few bytes to create the sha1
	apiURL := fmt.Sprintf("/v2/apps/%s/download", appGUID)
	request, err := http.NewRequest("GET", repo.config.APIEndpoint()+apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("%s: %s", T("Error building request"), err.Error())
	}
	request.Header.Set("Authorization", repo.config.AccessToken())
	request.Header.Set("accept", "application/json")
	request.Header.Set("Connection", "close")
	request.Header.Set("content-type", "application/json")
	request.Header.Set("User-Agent", "go-cli "+repo.config.CLIVersion()+" / "+runtime.GOOS)

	tr := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: repo.config.IsSSLDisabled()},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   2 * time.Second,
	}

	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	sha1, err := GetSha1FromReader(resp.Body)
	if err != nil {
		return "", err
	}
	return sha1, nil
}
func (repo CloudControllerApplicationBitsRepository) CopyBits(origAppGuid string, newAppGuid string) error {
	apiURL := fmt.Sprintf("%s/v2/apps/%s/copy_bits", repo.config.APIEndpoint(), newAppGuid)
	data := bytes.NewReader([]byte(fmt.Sprintf(`{"source_app_guid":"%s"}`, origAppGuid)))
	req, err := repo.gateway.NewRequest("POST", apiURL, repo.config.AccessToken(), data)
	if err != nil {
		return err
	}
	var job Job
	_, err = repo.gateway.PerformRequestForJSONResponse(req, &job)
	if err != nil {
		return err
	}
	for {
		job, err := repo.getJob(job.Entity.GUID)
		if err != nil {
			return err
		}
		if job.Entity.Status == "finished" {
			return nil
		}
		if job.Entity.Status == "failed" {
			return fmt.Errorf(
				"Error %s, %s [code: %d]",
				job.Entity.ErrorDetails.ErrorCode,
				job.Entity.ErrorDetails.Description,
				job.Entity.ErrorDetails.Code,
			)
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}
func (repo CloudControllerApplicationBitsRepository) getJob(jobGuid string) (Job, error) {
	apiURL := fmt.Sprintf("%s/v2/jobs/%s", repo.config.APIEndpoint(), jobGuid)
	req, err := repo.gateway.NewRequest("GET", apiURL, repo.config.AccessToken(), nil)
	if err != nil {
		return Job{}, err
	}
	var job Job
	_, err = repo.gateway.PerformRequestForJSONResponse(req, &job)
	if err != nil {
		return Job{}, err
	}
	return job, nil
}
func (repo CloudControllerApplicationBitsRepository) UploadBits(appGUID string, zipFile io.ReadCloser, fileSize int64) (apiErr error) {
	apiURL := fmt.Sprintf("/v2/apps/%s/bits", appGUID)
	fileutils.TempFile("requests", func(requestFile *os.File, err error) {
		if err != nil {
			apiErr = fmt.Errorf("%s: %s", T("Error creating tmp file: {{.Err}}", map[string]interface{}{"Err": err}), err.Error())
			return
		}

		presentFiles := []resources.AppFileResource{}

		presentFilesJSON, err := json.Marshal(presentFiles)
		if err != nil {
			apiErr = fmt.Errorf("%s: %s", T("Error marshaling JSON"), err.Error())
			return
		}

		boundary, err := repo.writeUploadBody(zipFile, fileSize, requestFile, presentFilesJSON)
		if err != nil {
			apiErr = fmt.Errorf("%s: %s", T("Error writing to tmp file: {{.Err}}", map[string]interface{}{"Err": err}), err.Error())
			return
		}

		var request *net.Request
		request, apiErr = repo.gateway.NewRequestForFile("PUT", repo.config.APIEndpoint()+apiURL, repo.config.AccessToken(), requestFile)
		if apiErr != nil {
			return
		}

		contentType := fmt.Sprintf("multipart/form-data; boundary=%s", boundary)
		request.HTTPReq.Header.Set("Content-Type", contentType)

		response := &resources.Resource{}
		_, apiErr = repo.gateway.PerformPollingRequestForJSONResponse(repo.config.APIEndpoint(), request, response, DefaultAppUploadBitsTimeout)
		if apiErr != nil {
			return
		}
	})

	return
}
func (repo CloudControllerApplicationBitsRepository) writeUploadBody(zipFile io.ReadCloser, fileSize int64, body *os.File, presentResourcesJSON []byte) (boundary string, err error) {
	writer := multipart.NewWriter(body)
	defer writer.Close()

	boundary = writer.Boundary()

	part, err := writer.CreateFormField("resources")
	if err != nil {
		return
	}
	_, err = io.Copy(part, bytes.NewBuffer(presentResourcesJSON))
	if err != nil {
		return
	}

	if zipFile != nil {

		part, zipErr := createZipPartWriter(fileSize, writer)
		if zipErr != nil {
			return
		}

		_, zipErr = io.Copy(part, zipFile)
		if zipErr != nil {
			return
		}
	}

	return
}

func createZipPartWriter(fileSize int64, writer *multipart.Writer) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="application"; filename="application.zip"`)
	h.Set("Content-Type", "application/zip")
	h.Set("Content-Length", fmt.Sprintf("%d", fileSize))
	h.Set("Content-Transfer-Encoding", "binary")
	return writer.CreatePart(h)
}

//////////////////
// Not used for now, this is an intent to make an upload in full stream (no intermediate file)
func (repo CloudControllerApplicationBitsRepository) UploadBitsTmp(appGUID string, zipFile io.ReadCloser, fileSize int64) error {
	apiURL := fmt.Sprintf("/v2/apps/%s/bits", appGUID)
	buf := new(bytes.Buffer)
	io.Copy(buf, zipFile)
	panic(buf)
	r, w := io.Pipe()
	mpw := multipart.NewWriter(w)
	go func() {
		var err error
		defer mpw.Close()
		defer w.Close()
		part, err := mpw.CreateFormField("resources")
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(part, bytes.NewBuffer([]byte("[]")))
		if err != nil {
			panic(err)
		}
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="application"; filename="application.zip"`)
		h.Set("Content-Type", "application/zip")
		h.Set("Content-Length", fmt.Sprintf("%d", fileSize))
		h.Set("Content-Transfer-Encoding", "binary")

		part, err = mpw.CreatePart(h)
		if err != nil {
			panic(err)
		}
		if _, err = io.Copy(part, zipFile); err != nil {
			panic(err)
		}
	}()
	var request *net.Request
	request, err := repo.gateway.NewRequest("PUT", repo.config.APIEndpoint()+apiURL, repo.config.AccessToken(), nil)
	if err != nil {
		return err
	}
	contentType := fmt.Sprintf("multipart/form-data; boundary=%s", mpw.Boundary())
	request.HTTPReq.Header.Set("Content-Type", contentType)
	request.HTTPReq.ContentLength = int64(repo.predictPart(int64(fileSize)))
	request.HTTPReq.Body = r

	response := &resources.Resource{}
	_, err = repo.gateway.PerformPollingRequestForJSONResponse(repo.config.APIEndpoint(), request, response, DefaultAppUploadBitsTimeout)
	if err != nil {
		panic(err)
	}
	return nil
}

func (repo CloudControllerApplicationBitsRepository) predictPart(filesize int64) int64 {
	buf := new(bytes.Buffer)
	mpw := multipart.NewWriter(buf)

	defer mpw.Close()
	part, err := mpw.CreateFormField("resources")
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(part, bytes.NewBuffer([]byte("[]")))
	if err != nil {
		panic(err)
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="application"; filename="application.zip"`)
	h.Set("Content-Type", "application/zip")
	h.Set("Content-Length", fmt.Sprintf("%d", filesize))
	h.Set("Content-Transfer-Encoding", "binary")

	part, err = mpw.CreatePart(h)
	b, _ := ioutil.ReadAll(buf)
	return int64(len(b)) + filesize
}

// end of the try
//////////////////
