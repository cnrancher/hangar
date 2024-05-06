package harbor

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/common/pkg/retry"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
)

var (
	ErrRegistryIsNotHarbor = errors.New("registry server is not harbor V2")
)

func GetURL(
	ctx context.Context,
	registry string,
	tlsVerify bool,
) (string, error) {
	client := &http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !tlsVerify},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	// Try ping registry using HTTPS protocol.
	registry = strings.TrimSuffix(registry, "/")
	u := fmt.Sprintf("https://%s/api/v2.0/ping", registry)
	ubase := fmt.Sprintf("https://%s", registry)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("harbor.GetURL: %w", err)
	}

	pingFunc := func() (*http.Response, error) {
		resp, err := utils.HTTPClientDo(ctx, client, req)
		if err == nil {
			defer resp.Body.Close()
			return resp, nil
		}
		if tlsVerify {
			return resp, err
		}

		logrus.Debugf("ping %s: %v", u, err)
		// The tlsVerify not enabled, try re-ping registry using HTTP.
		u = fmt.Sprintf("http://%s/api/v2.0/ping", registry)
		ubase = fmt.Sprintf("http://%s", registry)
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		resp, err = utils.HTTPClientDo(ctx, client, req)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()
		logrus.Debugf("ping %s: %v", u, resp.Status)
		return resp, nil
	}

	var resp *http.Response
	err = retry.IfNecessary(ctx, func() error {
		resp, err = pingFunc()
		return err
	}, &retry.Options{
		MaxRetry: 3,
		Delay:    time.Millisecond,
	})
	if err != nil {
		return "", fmt.Errorf("harbor.GetURL: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return ubase, nil
	}

	return "", ErrRegistryIsNotHarbor
}

// ProjectExists check project exists or not on harbor v2.
func ProjectExists(
	ctx context.Context,
	name, u string,
	credential *types.DockerAuthConfig,
	tlsVerify bool,
) (bool, error) {
	client := &http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !tlsVerify},
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	u = strings.TrimSuffix(u, "/")
	u = fmt.Sprintf("%s/api/v2.0/projects?project_name=%s", u, url.QueryEscape(name))
	r, err := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
	if err != nil {
		return false, fmt.Errorf("harbor.ProjectExists: %w", err)
	}
	auth := fmt.Sprintf("%s:%s", credential.Username, credential.Password)
	r.Header.Add("Authorization", "Basic "+utils.Base64(auth))
	r.Header.Add("Accept", "application/json")
	resp, err := utils.HTTPClientDoWithRetry(ctx, client, r)
	if err != nil {
		return false, fmt.Errorf("harbor.ProjectExists: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()
	switch resp.StatusCode {
	case http.StatusOK:
		logrus.Debugf("harbor project %q already exists", name)
		return true, nil
	case http.StatusNotFound:
		logrus.Debugf("harbor project %q not found", name)
	default:
		return false, fmt.Errorf("harbor.ProjectExists: %q response: %v",
			u, resp.Status)
	}

	return false, nil
}

// CreateProject creates project for harbor v2
func CreateProject(
	ctx context.Context,
	name, u string,
	credential *types.DockerAuthConfig,
	tlsVerify bool,
) error {
	data := struct {
		ProjectName string `json:"project_name"`
		Metadata    struct {
			Public string `json:"public"`
		} `json:"metadata"`
	}{
		ProjectName: name,
		Metadata: struct {
			Public string `json:"public"`
		}{
			Public: "false",
		},
	}
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("harbor.CreateHarborProject: json.Marshal: %w", err)
	}

	client := &http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !tlsVerify},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	u = strings.TrimSuffix(u, "/")
	u = fmt.Sprintf("%s/api/v2.0/projects", u)
	r, err := http.NewRequestWithContext(
		ctx, http.MethodPost, u, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("harbor.CreateProject: %w", err)
	}
	auth := fmt.Sprintf("%s:%s", credential.Username, credential.Password)
	r.Header.Add("Authorization", "Basic "+utils.Base64(auth))
	r.Header.Add("Content-Type", "application/json")
	resp, err := utils.HTTPClientDoWithRetry(ctx, client, r)
	if err != nil {
		return fmt.Errorf("harbor.CreateProject: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()
	switch resp.StatusCode {
	case http.StatusCreated:
	case http.StatusConflict:
		logrus.Debugf("already created project %q, response: %s",
			name, resp.Status)
	default:
		return fmt.Errorf("failed to create project %q, response: %s",
			name, resp.Status)
	}
	return nil
}
