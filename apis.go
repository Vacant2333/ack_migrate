package main

import (
	alibabacloudproviderv1alpha1 "github.com/cloudpilot-ai/lib/pkg/alibabacloud/karpenter-provider-alibabacloud/apis/v1alpha1"
	alibabacloudcorev1 "github.com/cloudpilot-ai/lib/pkg/alibabacloud/karpenter/apis/v1"
)

type RebalanceNodePoolList struct {
	ECSNodePools []ECSNodePool `json:"ecsNodePools"`
}

type RebalanceNodeClassList struct {
	ECSNodeClasses []ECSNodeClass `json:"ecsNodeClasses"`
}

type RebalanceNodePool struct {
	ECSNodePool *ECSNodePool `json:"ecsNodePool"`
}

type RebalanceNodeClass struct {
	ECSNodeClass *ECSNodeClass `json:"ecsNodeClass"`
}

type ECSNodePool struct {
	Name         string                           `json:"name"`
	Enable       bool                             `json:"enable"`
	NodePoolSpec *alibabacloudcorev1.NodePoolSpec `json:"nodePoolSpec"`
}

type ECSNodeClass struct {
	Name          string                                         `json:"name"`
	NodeClassSpec *alibabacloudproviderv1alpha1.ECSNodeClassSpec `json:"nodeClassSpec"`
}
