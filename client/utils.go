package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// helper function for all requests
func clientRequest(reqType string, apiURL string, apiToken string, payload interface{}) (*http.Response, error) {
	var jsonData []byte
	var err error

	if payload != nil {
		jsonData, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("error marshaling payload: %v", err)
		}
	}
	req, err := http.NewRequest(reqType, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}
