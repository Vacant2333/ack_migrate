package api

import (
	alibabacloudproviderv1alpha1 "github.com/cloudpilot-ai/lib/pkg/alibabacloud/karpenter-provider-alibabacloud/apis/v1alpha1"
	alibabacloudcorev1 "github.com/cloudpilot-ai/lib/pkg/alibabacloud/karpenter/apis/v1"
	awsproviderv1 "github.com/cloudpilot-ai/lib/pkg/aws/karpenter-provider-aws/apis/v1"
	awscorev1 "github.com/cloudpilot-ai/lib/pkg/aws/karpenter/apis/v1"
)

type RegisterClusterRequest struct {
	Demo          bool           `json:"demo"`
	AgentVersion  string         `json:"agentVersion"`
	CloudProvider string         `json:"cloudProvider"`
	GPUInstances  []string       `json:"gpuInstances"`
	Arch          []string       `json:"arch"`
	EKS           *EKSParams     `json:"eks"`
	ClusterParams *ClusterParams `json:"clusterParams"`
}

type ClusterParams struct {
	ClusterName    string `json:"clusterName"`
	ClusterVersion string `json:"clusterVersion"`
	Region         string `json:"region"`
	AccountID      string `json:"accountId"`
}

type EKSParams struct {
	ClusterName    string `json:"clusterName"`
	ClusterVersion string `json:"clusterVersion"`
	Region         string `json:"region"`
	AccountID      string `json:"accountId"`
}

type RegisterClusterResponse struct {
	ClusterID string `json:"clusterId"`
}

type ResponseBody struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Copy from github.com/cloudpilot-ai/cloudpilot/pkg/apiserver/database/serializer.go

type RebalanceConfig struct {
	// AWS
	EC2NodePools   []EC2NodePool  `json:"ec2NodePools,omitempty"`
	EC2NodeClasses []EC2NodeClass `json:"ec2NodeClasses,omitempty"`

	// AlibabaCloud
	ECSNodePools   []ECSNodePool  `json:"ecsNodePools,omitempty"`
	ECSNodeClasses []ECSNodeClass `json:"ecsNodeClasses,omitempty"`
}

type EC2NodePool struct {
	// +required
	Name string `json:"name"`
	// +required
	Enable bool `json:"enable"`
	// TODO: Used for compatible in v0.37.7, delete it latter
	// +optional
	NodePoolAnnotation map[string]string `json:"nodePoolAnnotation"`
	// +required
	NodePoolSpec *awscorev1.NodePoolSpec `json:"nodePoolSpec"`
}

type EC2NodeClass struct {
	// +required
	Name string `json:"name"`
	// TODO: Used for compatible in v0.37.7, delete it latter
	// +optional
	NodeClassAnnotation map[string]string `json:"nodeClassAnnotation"`
	// +required
	NodeClassSpec *awsproviderv1.EC2NodeClassSpec `json:"nodeClassSpec"`
}

type ECSNodePool struct {
	// +required
	Name string `json:"name"`
	// +required
	Enable bool `json:"enable"`
	// +required
	NodePoolSpec *alibabacloudcorev1.NodePoolSpec `json:"nodePoolSpec"`
}

type ECSNodeClass struct {
	// +required
	Name string `json:"name"`
	// +required
	NodeClassSpec *alibabacloudproviderv1alpha1.ECSNodeClassSpec `json:"nodeClassSpec"`
}

// Copy from github.com/cloudpilot-ai/cloudpilot/pkg/apiserver/apis/rebalance.go

type RebalanceNodePool struct {
	// AWS
	EC2NodePool *EC2NodePool `json:"ec2NodePool"`
	// Alibaba Cloud
	ECSNodePool *ECSNodePool `json:"ecsNodePool"`
}

type RebalanceNodePoolList struct {
	// AWS
	EC2NodePools []EC2NodePool `json:"ec2NodePools"`
	// Alibaba Cloud
	ECSNodePools []ECSNodePool `json:"ecsNodePools"`
}

type RebalanceNodeClass struct {
	// AWS
	EC2NodeClass *EC2NodeClass `json:"ec2NodeClass"`
	// Alibaba Cloud
	ECSNodeClass *ECSNodeClass `json:"ecsNodeClass"`
}

type RebalanceNodeClassList struct {
	// AWS
	EC2NodeClasses []EC2NodeClass `json:"ec2NodeClasses"`
	// Alibaba Cloud
	ECSNodeClasses []ECSNodeClass `json:"ecsNodeClasses"`
}
