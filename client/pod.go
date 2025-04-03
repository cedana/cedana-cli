package client

import (
	"encoding/json"
	"fmt"

	"github.com/cedana/cedana-cli/pkg/config"
)

// GetClusterNodes makes a POST request to fetch nodes for a given cluster
func GetClusterPods(clusterName string, clusterNamespace string) ([]Pod, error) {
	cedanaURL := config.Global.Connection.URL
	cedanaAuthToken := config.Global.Connection.AuthToken
  
	payload := map[string]string{
		"cluster_name": clusterName,
	}
  
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %v", err)
	}
  
	resp, err := clientRequest("POST", cedanaURL+"/cluster/pods/"+clusterNamespace, cedanaAuthToken, jsonData)
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
