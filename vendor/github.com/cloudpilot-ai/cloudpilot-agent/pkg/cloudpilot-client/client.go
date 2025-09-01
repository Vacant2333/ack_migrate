package cloudpilot_client

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"github.com/cloudpilot-ai/cloudpilot-agent/pkg/cloudpilot-client/api"
	"github.com/cloudpilot-ai/cloudpilot-agent/pkg/utils/leveledlogger"
	"github.com/cloudpilot-ai/cloudpilot-agent/pkg/values"
)

type Interface interface {
	RegisterCluster(req api.RegisterClusterRequest) (string, error)
	SendHeartBeat() error

	SendClusterDeltas(encodeData []byte) error
	SendOptimizationExpectation(encodeData []byte) error
	SendClusterSpotEvents(events []api.SpotEvent) error
	SendEventData(data []byte, urlPath string) error

	GetClusterRebalanceStatus() (api.ClusterRebalanceStatus, error)
	UpdateClusterRebalanceStatus(state api.ClusterRebalanceState, msg string) error

	GetClusterRebalanceConfiguration() (api.ClusterRebalanceConfiguration, error)
	UpdateClusterRebalanceConfiguration(cfg api.ClusterRebalanceConfiguration) error
	GetWorkloadRebalanceConfiguration() (api.WorkloadRebalanceConfiguration, error)

	UpdateClusterBilling(detail api.ClusterBilling) error
	GetClusterCostSummary() (api.ClusterCostsSummary, error)

	GetClusterID() string

	ListClusterRebalanceNodeClasses() (api.RebalanceNodeClassList, error)
	UpdateClusterRebalanceNodeClass(cloudProvider string, nodeClass api.RebalanceNodeClass) error
	ListClusterRebalanceNodePools() (api.RebalanceNodePoolList, error)
	UpdateClusterRebalanceNodePool(cloudProvider string, nodePool api.RebalanceNodePool) error
}

type Client struct {
	API       string
	APIKEY    string
	ClusterID string
}

func NewCloudPilotClient(api, apiKey, clusterID string) Interface {
	return &Client{
		API:       api,
		APIKEY:    apiKey,
		ClusterID: clusterID,
	}
}

func (c *Client) RegisterCluster(req api.RegisterClusterRequest) (string, error) {
	url := fmt.Sprintf("%s/api/v1/clusters/registration", c.API)
	resp, err := c.request(http.MethodPost, url, req)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return "", err
	}

	var stdResp api.ResponseBody
	if err := json.Unmarshal(body, &stdResp); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to register cluster: %v", stdResp.Message)
		return "", fmt.Errorf("failed to register cluster: %v", stdResp.Message)
	}
	dataBytes, err := json.Marshal(stdResp.Data)
	if err != nil {
		klog.Errorf("Failed to marshal data: %v", err)
		return "", err
	}

	registerResponse := &api.RegisterClusterResponse{}
	if err := json.Unmarshal(dataBytes, registerResponse); err != nil {
		klog.Errorf("Failed to unmarshal data to registerResponse: %v", err)
		return "", err
	}
	c.ClusterID = registerResponse.ClusterID
	return registerResponse.ClusterID, nil
}

func (c *Client) SendHeartBeat() error {
	url := fmt.Sprintf("%s/api/v1/clusters/%s/heartbeat", c.API, c.ClusterID)
	resp, err := c.request(http.MethodGet, url, nil)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to send heartbeat: %v", resp.Status)
		return fmt.Errorf("failed to send heartbeat: %v", resp.Status)
	}
	return err
}

func (c *Client) SendClusterDeltas(encodeData []byte) error {
	// TBD: we may compress the data to send smaller size of data
	url := fmt.Sprintf("%s/api/v1/clusters/%s/deltas", c.API, c.ClusterID)
	httpReq, err := c.newHTTPReq(http.MethodPost, url, encodeData)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	client := retryablehttp.NewClient()
	client.Logger = leveledlogger.NewKlogLeveledLogger()
	resp, err := client.Do(httpReq)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			klog.Errorf("Failed to read response body: %v", err)
			return err
		}
		standResp := &api.ResponseBody{}
		_ = json.Unmarshal(body, standResp)
		klog.Errorf("Failed to send cluster deltas: %v", standResp.Message)
		return fmt.Errorf("failed to send cluster deltas: %v", standResp.Message)
	}
	return nil
}

func (c *Client) SendOptimizationExpectation(encodeData []byte) error {
	url := fmt.Sprintf("%s/api/v1/clusters/%s/optimization", c.API, c.ClusterID)
	httpReq, err := c.newHTTPReq(http.MethodPost, url, encodeData)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	client := retryablehttp.NewClient()
	client.Logger = leveledlogger.NewKlogLeveledLogger()
	resp, err := client.Do(httpReq)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			klog.Errorf("Failed to read response body: %v", err)
			return err
		}
		standResp := &api.ResponseBody{}
		_ = json.Unmarshal(body, standResp)
		klog.Errorf("Failed to send cluster optimization expectation: %v", standResp.Message)
		return fmt.Errorf("failed to send cluster optimization expectationtas: %v", standResp.Message)
	}
	return nil
}

func (c *Client) GetClusterRebalanceConfiguration() (api.ClusterRebalanceConfiguration, error) {
	url := fmt.Sprintf("%s/api/v1/rebalance/clusters/%s/configuration", c.API, c.ClusterID)
	resp, err := c.request(http.MethodGet, url, nil)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return api.ClusterRebalanceConfiguration{}, err
	}
	defer resp.Body.Close()

	respText, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return api.ClusterRebalanceConfiguration{}, err
	}

	var stdResp api.ResponseBody
	if err := json.Unmarshal(respText, &stdResp); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return api.ClusterRebalanceConfiguration{}, err
	}

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to get provider configuration: %v", stdResp.Message)
		return api.ClusterRebalanceConfiguration{}, fmt.Errorf("failed to provider configuration: %v", stdResp.Message)
	}

	dataBytes, err := json.Marshal(stdResp.Data)
	if err != nil {
		klog.Errorf("Failed to marshal data: %v", err)
		return api.ClusterRebalanceConfiguration{}, err
	}

	var cfg api.ClusterRebalanceConfiguration
	if err := json.Unmarshal(dataBytes, &cfg); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return api.ClusterRebalanceConfiguration{}, err
	}

	return cfg, nil
}

func (c *Client) ListClusterRebalanceNodePools() (api.RebalanceNodePoolList, error) {
	url := fmt.Sprintf("%s/api/v1/rebalance/clusters/%s/nodepools", c.API, c.ClusterID)
	resp, err := c.request(http.MethodGet, url, nil)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return api.RebalanceNodePoolList{}, err
	}
	defer resp.Body.Close()

	respText, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return api.RebalanceNodePoolList{}, err
	}

	var stdResp api.ResponseBody
	if err := json.Unmarshal(respText, &stdResp); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return api.RebalanceNodePoolList{}, err
	}

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to get nodepools: %v", stdResp.Message)
		return api.RebalanceNodePoolList{}, fmt.Errorf("failed to nodepools: %v", stdResp.Message)
	}

	dataBytes, err := json.Marshal(stdResp.Data)
	if err != nil {
		klog.Errorf("Failed to marshal data: %v", err)
		return api.RebalanceNodePoolList{}, err
	}

	var nodePoolList api.RebalanceNodePoolList
	if err := json.Unmarshal(dataBytes, &nodePoolList); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return api.RebalanceNodePoolList{}, err
	}

	return nodePoolList, nil
}

func (c *Client) UpdateClusterRebalanceNodePool(cloudProvider string, nodePool api.RebalanceNodePool) error {
	var url string
	switch cloudProvider {
	case values.CloudProviderAWS:
		url = fmt.Sprintf("%s/api/v1/rebalance/clusters/%s/nodepools", c.API, c.ClusterID)
	default:
		klog.Errorf("Unknown cloud provider: %s", cloudProvider)
		return fmt.Errorf("unknown cloud provider: %s", cloudProvider)
	}

	resp, err := c.request(http.MethodPost, url, nodePool)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to update node pool: %v", resp.Status)
		return fmt.Errorf("failed to update node pool: %v", resp.Status)
	}

	return nil
}

func (c *Client) ListClusterRebalanceNodeClasses() (api.RebalanceNodeClassList, error) {
	url := fmt.Sprintf("%s/api/v1/rebalance/clusters/%s/nodeclasses", c.API, c.ClusterID)
	resp, err := c.request(http.MethodGet, url, nil)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return api.RebalanceNodeClassList{}, err
	}
	defer resp.Body.Close()

	respText, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return api.RebalanceNodeClassList{}, err
	}

	var stdResp api.ResponseBody
	if err := json.Unmarshal(respText, &stdResp); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return api.RebalanceNodeClassList{}, err
	}

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to get nodeclasses: %v", stdResp.Message)
		return api.RebalanceNodeClassList{}, fmt.Errorf("failed to nodeclasses: %v", stdResp.Message)
	}

	dataBytes, err := json.Marshal(stdResp.Data)
	if err != nil {
		klog.Errorf("Failed to marshal data: %v", err)
		return api.RebalanceNodeClassList{}, err
	}

	var nodeClassList api.RebalanceNodeClassList
	if err := json.Unmarshal(dataBytes, &nodeClassList); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return api.RebalanceNodeClassList{}, err
	}

	return nodeClassList, nil
}

func (c *Client) UpdateClusterRebalanceNodeClass(cloudProvider string, nodeClass api.RebalanceNodeClass) error {
	var url string
	switch cloudProvider {
	case values.CloudProviderAWS:
		url = fmt.Sprintf("%s/api/v1/rebalance/clusters/%s/nodeclasses", c.API, c.ClusterID)
	default:
		klog.Errorf("Unknown cloud provider: %s", cloudProvider)
		return fmt.Errorf("unknown cloud provider: %s", cloudProvider)
	}

	resp, err := c.request(http.MethodPost, url, nodeClass)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to update node class: %v", resp.Status)
		return fmt.Errorf("failed to update node class: %v", resp.Status)
	}

	return nil
}

func (c *Client) UpdateClusterRebalanceConfiguration(cfg api.ClusterRebalanceConfiguration) error {
	url := fmt.Sprintf("%s/api/v1/rebalance/clusters/%s/configuration", c.API, c.ClusterID)
	resp, err := c.request(http.MethodPost, url, cfg)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return err
	}
	defer resp.Body.Close()

	respText, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return err
	}

	var stdResp api.ResponseBody
	if err := json.Unmarshal(respText, &stdResp); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to get provider configuration: %v", stdResp.Message)
		return fmt.Errorf("failed to provider configuration: %v", stdResp.Message)
	}
	return nil
}

func (c *Client) UpdateClusterRebalanceStatus(state api.ClusterRebalanceState, msg string) error {
	status := api.ClusterRebalanceStatus{
		State:                    state,
		LastComponentsActiveTime: metav1.Now(),
		Message:                  msg,
	}
	url := fmt.Sprintf("%s/api/v1/rebalance/clusters/%s/status", c.API, c.ClusterID)
	resp, err := c.request(http.MethodPost, url, status)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to send cluster rebalance status: %v", resp.StatusCode)
		return fmt.Errorf("failed to send cluster rebalance status: %v", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetClusterRebalanceStatus() (api.ClusterRebalanceStatus, error) {
	url := fmt.Sprintf("%s/api/v1/rebalance/clusters/%s/status", c.API, c.ClusterID)
	resp, err := c.request(http.MethodGet, url, nil)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return api.ClusterRebalanceStatus{}, err
	}
	defer resp.Body.Close()

	respText, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return api.ClusterRebalanceStatus{}, err
	}

	var stdResp api.ResponseBody
	if err := json.Unmarshal(respText, &stdResp); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return api.ClusterRebalanceStatus{}, err
	}

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to get provider status: %v", stdResp.Message)
		return api.ClusterRebalanceStatus{}, fmt.Errorf("failed to get provider status: %v", stdResp.Message)
	}
	dataBytes, err := json.Marshal(stdResp.Data)
	if err != nil {
		klog.Errorf("Failed to marshal data: %v", err)
		return api.ClusterRebalanceStatus{}, fmt.Errorf("failed to marshal data: %v", err)
	}

	var status api.ClusterRebalanceStatus
	if err := json.Unmarshal(dataBytes, &status); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return api.ClusterRebalanceStatus{}, err
	}

	return status, nil
}

func (c *Client) GetWorkloadRebalanceConfiguration() (api.WorkloadRebalanceConfiguration, error) {
	url := fmt.Sprintf("%s/api/v1/rebalance/clusters/%s/workloads/configuration", c.API, c.ClusterID)
	resp, err := c.request(http.MethodGet, url, nil)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return api.WorkloadRebalanceConfiguration{}, err
	}
	defer resp.Body.Close()

	respText, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return api.WorkloadRebalanceConfiguration{}, err
	}

	var stdResp api.ResponseBody
	if err := json.Unmarshal(respText, &stdResp); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return api.WorkloadRebalanceConfiguration{}, err
	}

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to get provider configuration: %v", stdResp.Message)
		return api.WorkloadRebalanceConfiguration{}, fmt.Errorf("failed to get provider configuration: %v", stdResp.Message)
	}
	dataBytes, err := json.Marshal(stdResp.Data)
	if err != nil {
		klog.Errorf("Failed to marshal data: %v", err)
		return api.WorkloadRebalanceConfiguration{}, err
	}

	var cfg api.WorkloadRebalanceConfiguration
	if err := json.Unmarshal(dataBytes, &cfg); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return api.WorkloadRebalanceConfiguration{}, err
	}

	return cfg, nil
}

func (c *Client) UpdateClusterBilling(billing api.ClusterBilling) error {
	url := fmt.Sprintf("%s/api/v1/clusters/%s/billing", c.API, c.ClusterID)
	resp, err := c.request(http.MethodPost, url, billing)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to update cluster billing: %v", resp.Status)
		return fmt.Errorf("failed to update cluster billing: %v", resp.Status)
	}
	return nil
}

func (c *Client) SendClusterSpotEvents(events []api.SpotEvent) error {
	url := fmt.Sprintf("%s/api/v1/clusters/%s/events/spot", c.API, c.ClusterID)
	resp, err := c.request(http.MethodPost, url, events)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to send cluster spot events: %v", resp.Status)
		return fmt.Errorf("failed to send cluster spot events: %v", resp.Status)
	}
	return nil
}

func (c *Client) GetClusterCostSummary() (api.ClusterCostsSummary, error) {
	url := fmt.Sprintf("%s/api/v1/costs/clusters/%s/summary", c.API, c.ClusterID)
	resp, err := c.request(http.MethodGet, url, nil)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return api.ClusterCostsSummary{}, err
	}
	defer resp.Body.Close()

	respText, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return api.ClusterCostsSummary{}, err
	}

	var stdResp api.ResponseBody
	if err := json.Unmarshal(respText, &stdResp); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return api.ClusterCostsSummary{}, err
	}

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to get cluster cost summary: %v", resp.Status)
		return api.ClusterCostsSummary{}, fmt.Errorf("failed to get cluster cost summary: %v", resp.Status)
	}
	dataBytes, err := json.Marshal(stdResp.Data)
	if err != nil {
		klog.Errorf("Failed to marshal data: %v", err)
		return api.ClusterCostsSummary{}, err
	}

	var costs api.ClusterCostsSummary
	if err := json.Unmarshal(dataBytes, &costs); err != nil {
		klog.Errorf("Failed to unmarshal response body: %v", err)
		return api.ClusterCostsSummary{}, err
	}
	return costs, nil
}

func (c *Client) SendEventData(data []byte, urlPath string) error {
	url := fmt.Sprintf(`%s`+urlPath, c.API, c.ClusterID)
	resp, err := c.requestData(http.MethodPost, url, data)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Failed to send event data: %v", resp.Status)
		return fmt.Errorf("failed to send event data: %v", resp.Status)
	}

	return nil
}

func (c *Client) request(method string, url string, reqBody any) (*http.Response, error) {
	var httpReq *retryablehttp.Request
	if reqBody != nil {
		reqBodyJson, err := json.Marshal(reqBody)
		if err != nil {
			klog.Errorf("Failed to marshal request body: %v", err)
			return nil, err
		}
		httpReq, err = c.newHTTPReq(method, url, reqBodyJson)
		if err != nil {
			klog.Errorf("Failed to create http request: %v", err)
			return nil, err
		}
	} else {
		var err error
		httpReq, err = c.newHTTPReq(method, url, nil)
		if err != nil {
			klog.Errorf("Failed to create http request: %v", err)
			return nil, err
		}
	}
	client := retryablehttp.NewClient()
	client.Logger = leveledlogger.NewKlogLeveledLogger()

	resp, err := client.Do(httpReq)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return nil, err
	}
	return resp, nil
}

func (c *Client) requestData(method string, url string, data []byte) (resp *http.Response, err error) {
	var (
		httpReq        *retryablehttp.Request
		compressedData bytes.Buffer
	)
	if len(data) == 0 {
		return nil, fmt.Errorf("data is empty")
	}

	gz := gzip.NewWriter(&compressedData)
	if _, err := gz.Write(data); err != nil {
		klog.Errorf("Failed to compress request body: %v", err)
		return nil, err
	}
	if err := gz.Close(); err != nil {
		klog.Errorf("Failed to close gzip writer: %v", err)
		return nil, err
	}

	httpReq, err = c.newHTTPReq(method, url, compressedData.Bytes())
	if err != nil {
		klog.Errorf("Failed to create http request: %v", err)
		return nil, err
	}
	httpReq.Header.Set("Content-Encoding", "gzip")

	client := retryablehttp.NewClient()
	client.Logger = leveledlogger.NewKlogLeveledLogger()

	resp, err = client.Do(httpReq)
	if err != nil {
		klog.Errorf("Failed to send http request: %v", err)
		return nil, err
	}

	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			klog.Errorf("Failed to create gzip reader: %v", err)
			return nil, err
		}
		defer reader.Close()
		body, err := io.ReadAll(reader)
		if err != nil {
			klog.Errorf("Failed to read response body: %v", err)
			return nil, err
		}
		resp.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	return resp, nil
}

func (c *Client) newHTTPReq(method, url string, body []byte) (*retryablehttp.Request, error) {
	httpReq, err := retryablehttp.NewRequest(method, url, body)
	if err != nil {
		klog.Errorf("Failed to create http request: %v", err)
		return nil, err
	}
	httpReq.Header.Set("X-API-KEY", c.APIKEY)
	return httpReq, nil
}

func (c *Client) GetClusterID() string {
	return c.ClusterID
}
