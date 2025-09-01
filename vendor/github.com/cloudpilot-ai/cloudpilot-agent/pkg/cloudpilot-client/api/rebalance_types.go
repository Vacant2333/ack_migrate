package api

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterRebalanceConfiguration struct {
	// UploadConfig represents whether the config uploaded from agent.
	// When Karpenter is upgraded, the config may be changed by Karpenter, it should be uploaded to server side.
	// If we do the conversion in server side, it's supper complex, so, we choose this way.
	// +optional
	UploadConfig bool `json:"uploadConfig"`
	// +required
	Enable bool `json:"enable"`
	// +optional
	EnableDiversityInstanceType bool `json:"enableDiversityInstanceType"`
}

type ClusterRebalanceState string

const (
	ClusterRebalanceStateApplying              ClusterRebalanceState = "Applying"
	ClusterRebalanceStateLaunchingReplacements ClusterRebalanceState = "LaunchingReplacements"
	ClusterRebalanceStateDraining              ClusterRebalanceState = "Draining"
	ClusterRebalanceStateTerminating           ClusterRebalanceState = "Terminating"
	ClusterRebalanceStateFailed                ClusterRebalanceState = "Failed"
	ClusterRebalanceStateSuccess               ClusterRebalanceState = "Success"
)

type ClusterRebalanceStatus struct {
	// +optional
	State ClusterRebalanceState `json:"state"`
	// +optional
	LastComponentsActiveTime metav1.Time `json:"lastComponentsActiveTime"`
	// +optional
	Message string `json:"message"`
}

type WorkloadRebalanceConfiguration struct {
	// +optional
	Workloads WorkloadsSlice `json:"workloads,omitempty"`
}

type (
	WorkloadsSlice []Workload
	WorkloadsSet   map[string]*Workload
)

type Workload struct {
	// +required
	Name string `json:"name,omitempty"`
	// +required
	Type string `json:"type,omitempty"`
	// +required
	Namespace string `json:"namespace"`
	// +required
	Replicas int32 `json:"replicas"`
	// +required
	RebalanceAble bool `json:"rebalanceAble"`
	// +optional
	SpotFriendly bool `json:"spotFriendly"`
	// +optional
	MinNonSpotReplicas int32 `json:"minNonSpotReplicas"`
}

func (c WorkloadsSlice) AsSet() WorkloadsSet {
	return lo.Associate(c, func(item Workload) (string, *Workload) {
		return GenerateWorkloadKey(item.Name, item.Type, item.Namespace), &item
	})
}

func GenerateWorkloadKey(workloadName, workloadType, namespace string) string {
	return fmt.Sprintf("%s/%s/%s", strings.ToLower(workloadType), namespace, workloadName)
}
