package client

// Node represents a node in the cluster response
type Node struct {
	ID           string `json:"ID"`
	ClusterID    string `json:"ClusterID"`
	Name         string `json:"Name"`
	Status       string `json:"Status"`
	ComputeType  string `json:"ComputeType"`
	InstanceType string `json:"InstanceType"`
	Region       string `json:"Region"`
}

// Cluster represents a cluster in the response
type Cluster struct {
	ID       string      `json:"ID"`
	OrgID    string      `json:"OrgID"`
	Name     string      `json:"Name"`
	Status   string      `json:"Status"`
	Metadata interface{} `json:"Metadata"`
}

type Pod struct {
	ID        string      `json:"ID"`
	ClusterID string      `json:"ClusterID"`
	NodeID    string      `json:"NodeID"`
	Name      string      `json:"Name"`
	Namespace string      `json:"Namespace"`
	Status    string      `json:"Status"`
	Metadata  interface{} `json:"Metadata"`
}
