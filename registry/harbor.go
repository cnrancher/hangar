package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

type HarborProjectTemplate struct {
	ProjectName string                        `json:"project_name"`
	Metadata    HarborProjectMetadataTemplate `json:"metadata"`
}

type HarborProjectMetadataTemplate struct {
	Public string `json:"public"`
}

// CreateHarborProject creates project for harbor v2
func CreateHarborProject(name, url, username, passwd string) error {
	// result_code=$(curl -k -s -u "${harbor_user}:${harbor_password}"
	//  -X POST -H "Content-type:application/json" -d
	// '{"project_name":"'"${project}"'","metadata":{"public":"true"}}' $url)
	values := HarborProjectTemplate{
		ProjectName: name,
		Metadata: HarborProjectMetadataTemplate{
			Public: "true",
		},
	}
	json_data, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("CreateHarborProject: json.Marshal: %w", err)
	}

	// URL-encoded payload
	client := &http.Client{}
	r, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(json_data))
	auth := fmt.Sprintf("%s:%s", username, passwd)
	r.Header.Add("Authorization", "Basic "+u.Base64(auth))
	r.Header.Add("Content-Type", "application/json")

	// send a json post request
	resp, err := client.Do(r)
	if err != nil {
		return fmt.Errorf("CreateHarborProject: %w", err)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	logrus.Debugf("Create %q response: %s", name, string(bodyBytes))
	logrus.Debugf("Status: %d", resp.StatusCode)
	return nil
}
