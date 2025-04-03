package client

import (
	"fmt"
	"io"
  "net/http"
	"github.com/cedana/cedana-cli/pkg/config"
)

// GetClusterNodes makes a POST request to fetch nodes for a given cluster
func CreateWorkload(payload interface{}) (string, error) {
	cedanaURL := config.Global.Connection.URL
	cedanaAuthToken := config.Global.Connection.AuthToken

	var resp *http.Response
	var err error

	if contentType == "yaml" {
		resp, err = yamlClientRequest("POST", cedanaURL+"/cluster/workload", cedanaAuthToken, payload)
	} else {
		resp, err = clientRequest("POST", cedanaURL+"/cluster/workload", cedanaAuthToken, payload)
	}

	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%v", err)
	}
	return string(bodyBytes), nil
}

func DeleteWorkload(payload []byte, contentType string) (string, error) {
  cedanaURL := config.Global.Connection.URL
	cedanaAuthToken := config.Global.Connection.AuthToken

	var resp *http.Response
	var err error

	if contentType == "yaml" {
		resp, err = yamlClientRequest("DELETE", cedanaURL+"/cluster/workload", cedanaAuthToken, payload)
	} else {
		resp, err = clientRequest("DELETE", cedanaURL+"/cluster/workload", cedanaAuthToken, payload)
	}
  
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
  
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%v", err)
	}
	return string(bodyBytes), nil
}
