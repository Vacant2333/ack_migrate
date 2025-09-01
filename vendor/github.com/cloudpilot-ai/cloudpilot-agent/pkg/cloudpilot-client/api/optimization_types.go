package api

import (
	v1 "k8s.io/api/core/v1"
)

type ResourceRate map[string]float64

type ClusterOptimization struct {
	Errors                       map[string]string  `json:"errors"`
	NodeOptimizations            []NodeOptimization `json:"nodeOptimizations"`
	ClusterAllocatedResourceRate ResourceRate       `json:"clusterAllocatedResourceRate"`
}

type NodeOptimization struct {
	Count        int    `json:"size"`
	CapacityType string `json:"capacityType"`
	InstanceType string `json:"instanceType"`
}

// CalculateResourceRate computes the resource allocation rate given total resources and requested resources.
func CalculateResourceRate(capacity, requested v1.ResourceList) (ResourceRate, error) {
	resourceRate := ResourceRate{}
	for res, capQty := range capacity {
		reqQty, exists := requested[res]
		if !exists {
			resourceRate[res.String()] = 0
			continue
		}
		if capQty.IsZero() {
			continue
		}
		rate := float64(reqQty.MilliValue()) / float64(capQty.MilliValue())
		resourceRate[res.String()] = rate
	}

	return resourceRate, nil
}
