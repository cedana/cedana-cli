package client

import (
	"fmt"
	"io"
)

// GetClusterNodes makes a POST request to fetch nodes for a given cluster
func CreateWorkload(payload interface{}) (string, error) {

	resp, err := clientRequest("POST", cedanaURL+"/cluster/workload", cedanaAuthToken, payload)
	if err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%v", err)
	}
	return string(bodyBytes), nil
}

// GetClusterNodes makes a POST request to fetch nodes for a given cluster
func DeleteWorkload(payload interface{}, cedanaURL string, cedanaAuthToken string) (string, error) {

	resp, err := clientRequest("DELETE", cedanaURL+"/cluster/workload", cedanaAuthToken, payload)
	if err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%v", err)
	}
	return string(bodyBytes), nil
}
