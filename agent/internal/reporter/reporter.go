package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"v2ray-dash/agent/internal/model"
)

type Client struct {
	serverID        string
	controlCenterURL string
	psk             string
	httpClient      *http.Client
}

func New(controlCenterURL, serverID, psk string) *Client {
	return &Client{
		serverID:        serverID,
		controlCenterURL: controlCenterURL,
		psk:             psk,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) ReportStatus(status *model.NodeStatus) error {
	url := fmt.Sprintf("%s/api/agent/heartbeat", c.controlCenterURL)

	body, err := json.Marshal(status)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PSK", c.psk)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	return nil
}