package client

import (
	"encoding/json"
	"fmt"
)

// ListClusters makes a GET request to fetch all clusters
func ListClusters(cedanaURL string, cedanaAuthToken string) ([]Cluster, error) {
	var clusters []Cluster
	resp, err := clientRequest("GET", cedanaURL+"/cluster", cedanaAuthToken, nil)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&clusters); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}
	return clusters, nil
}
