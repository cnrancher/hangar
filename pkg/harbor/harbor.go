package harbor

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
)

// ProjectExists check project exists or not on harbor v2
func ProjectExists(name, u, uname, passwd string) (bool, error) {
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	if !cmdconfig.GetBool("tls-verify") {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	u = fmt.Sprintf("%s?project_name=%s", u, url.QueryEscape(name))
	r, err := http.NewRequest(http.MethodHead, u, nil)
	if err != nil {
		return false, fmt.Errorf("CheckHarborProject: %w", err)
	}
	auth := fmt.Sprintf("%s:%s", uname, passwd)
	r.Header.Add("Authorization", "Basic "+utils.Base64(auth))
	r.Header.Add("Accept", "application/json")
	resp, err := client.Do(r)
	if err != nil {
		return false, fmt.Errorf("CheckHarborProject: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		logrus.Debugf("harbor project %q already exists", name)
		return true, nil
	case http.StatusNotFound:
		logrus.Debugf("harbor project %q not found", name)
	default:
		return false, fmt.Errorf("ProjectExists: %q response: %v",
			u, resp.Status)
	}

	return false, nil
}

// CreateProject creates project for harbor v2
func CreateProject(name, u, uname, passwd string) error {
	values := struct {
		ProjectName string `json:"project_name"`
		Metadata    struct {
			Public string `json:"public"`
		} `json:"metadata"`
	}{
		ProjectName: name,
		Metadata: struct {
			Public string `json:"public"`
		}{
			Public: "true",
		},
	}
	json_data, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("CreateHarborProject: json.Marshal: %w", err)
	}

	client := &http.Client{
		Timeout: time.Second * 5,
	}
	if !cmdconfig.GetBool("tls-verify") {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	r, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(json_data))
	if err != nil {
		return fmt.Errorf("CreateHarborProject: %w", err)
	}
	auth := fmt.Sprintf("%s:%s", uname, passwd)
	r.Header.Add("Authorization", "Basic "+utils.Base64(auth))
	r.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(r)
	if err != nil {
		return fmt.Errorf("CreateHarborProject: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusCreated:
		logrus.Infof("created harbor project %q, response: %s",
			name, resp.Status)
	case http.StatusConflict:
		logrus.Debugf("already created project %q, response: %s",
			name, resp.Status)
	default:
		logrus.Errorf("failed to create project %q, response: %s",
			name, resp.Status)
	}

	return nil
}
