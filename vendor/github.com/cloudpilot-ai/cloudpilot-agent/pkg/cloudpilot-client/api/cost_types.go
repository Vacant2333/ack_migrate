package api

import "time"

type ClusterCostsSummary struct {
	ID                          string        `json:"id"`
	Demo                        bool          `json:"demo"`
	Name                        string        `json:"clusterName"`
	CloudProvider               string        `json:"cloudProvider"`
	Region                      string        `json:"region"`
	InitialMonthlyCost          float64       `json:"initialMonthlyCost"`
	InitialOptimizedMonthlyCost float64       `json:"initialOptimizedMonthlyCost"`
	MonthlyCost                 float64       `json:"monthlyCost"`
	EstimateMonthlySaving       float64       `json:"estimateMonthlySaving"`
	NodesNumber                 int           `json:"nodesNumber"`
	OnDemandNodesNumber         int           `json:"onDemandNodesNumber"`
	SpotNodesNumber             int           `json:"spotNodesNumber"`
	CPUCores                    float64       `json:"cpuCores"`
	RAMGiBs                     float64       `json:"ramGiBs"`
	GPUCards                    float64       `json:"gpuCards"`
	NeedUpgrade                 bool          `json:"needUpgrade"`
	OnboardManifestVersion      string        `json:"onboardManifestVersion"`
	Status                      ClusterStatus `json:"status"`
	JoinTime                    time.Time     `json:"joinTime"`
}

type ClusterStatus string

const (
	ClusterStatusOnline  ClusterStatus = "online"
	ClusterStatusOffline ClusterStatus = "offline"
)
