package harbor

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func GetRegistryURL(
	ctx context.Context, registry string, tlsVerify bool,
) (string, error) {
	client := &http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !tlsVerify},
		},
	}
	// Try ping registry using HTTPS protocol.
	registry = strings.TrimSuffix(registry, "/")
	u := fmt.Sprintf("https://%s/api/v2.0/ping", registry)
	ubase := fmt.Sprintf("https://%s", registry)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("harbor.GetRegistryURL: %w", err)
	}
	resp, err := httpClientDoWithRetry(ctx, client, req)
	if err != nil {
		if tlsVerify {
			return "", fmt.Errorf("harbor.GetRegistryURL: %w", err)
		}

		if errors.Is(err, http.ErrSchemeMismatch) {
			logrus.Debugf("ping %s: %v", u, err)
			// The tlsVerify not enabled, try re-ping registry using HTTP.
			u = fmt.Sprintf("http://%s/api/v2.0/ping", registry)
			ubase = fmt.Sprintf("http://%s", registry)
			req, err = http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
			if err != nil {
				return "", fmt.Errorf("harbor.GetRegistryURL: %w", err)
			}
			resp, err = httpClientDoWithRetry(ctx, client, req)
			if err != nil {
				return "", fmt.Errorf("harbor.GetRegistryURL: %w", err)
			}
		} else {
			return "", fmt.Errorf("harbor.GetRegistryURL: %w", err)
		}
	}
	defer resp.Body.Close()
	logrus.Debugf("ping %s: %v", u, resp.Status)

	switch resp.StatusCode {
	case http.StatusOK:
		b, _ := io.ReadAll(resp.Body)
		if len(b) > 0 {
			if len(b) > 20 {
				b = b[:20]
			}
			logrus.Debugf("server response: %v", string(b))
			content := strings.ToLower(string(b))
			if strings.Contains(content, "pong") {
				return ubase, nil
			}
		}
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
	resp, err := httpClientDoWithRetry(ctx, client, r)
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
	json_data, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("harbor.CreateHarborProject: json.Marshal: %w", err)
	}

	client := &http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !tlsVerify},
		},
	}
	u = strings.TrimSuffix(u, "/")
	u = fmt.Sprintf("%s/api/v2.0/projects", u)
	r, err := http.NewRequestWithContext(
		ctx, http.MethodPost, u, bytes.NewReader(json_data))
	if err != nil {
		return fmt.Errorf("harbor.CreateProject: %w", err)
	}
	auth := fmt.Sprintf("%s:%s", credential.Username, credential.Password)
	r.Header.Add("Authorization", "Basic "+utils.Base64(auth))
	r.Header.Add("Content-Type", "application/json")
	resp, err := httpClientDoWithRetry(ctx, client, r)
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

func httpClientDoWithRetry(
	ctx context.Context, client *http.Client, req *http.Request,
) (*http.Response, error) {
	var resp *http.Response
	var err error
	err = retry.IfNecessary(ctx, func() error {
		logrus.Debugf("client.Do: %v", req.URL.String())
		resp, err = client.Do(req)
		return err
	}, &retry.Options{
		MaxRetry: 3,
		Delay:    time.Microsecond * 100,
	})
	return resp, err
}
