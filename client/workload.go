package client

import (
	"fmt"
	"io"
	"net/http"
)

// GetClusterNodes makes a POST request to fetch nodes for a given cluster
func CreateWorkload(payload []byte, cedanaURL string, cedanaAuthToken string, contentType string) (string, error) {
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

// GetClusterNodes makes a POST request to fetch nodes for a given cluster
func DeleteWorkload(payload []byte, cedanaURL string, cedanaAuthToken string, contentType string) (string, error) {
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
