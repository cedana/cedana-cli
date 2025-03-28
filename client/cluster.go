package client

import (
	"encoding/json"
	"fmt"

	"github.com/cedana/cedana-cli/pkg/config"
)

// ListClusters makes a GET request to fetch all clusters
func ListClusters() ([]Cluster, error) {
	cedanaURL := config.Global.Connection.URL
	cedanaAuthToken := config.Global.Connection.AuthToken
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
