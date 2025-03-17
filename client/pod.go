package client

import (
	"encoding/json"
	"fmt"
)

// GetClusterNodes makes a POST request to fetch nodes for a given cluster
func GetClusterPods(clusterName string, clusterNamespace string, cedanaURL string, cedanaAuthToken string) ([]Pod, error) {

	// Create request payload
	payload := map[string]string{
		"cluster_name": clusterName,
	}
	resp, err := clientRequest("POST", cedanaURL+"/cluster/pods/"+clusterNamespace, cedanaAuthToken, payload)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}
	defer resp.Body.Close()
	var pods []Pod
	if err := json.NewDecoder(resp.Body).Decode(&pods); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return pods, nil
}
