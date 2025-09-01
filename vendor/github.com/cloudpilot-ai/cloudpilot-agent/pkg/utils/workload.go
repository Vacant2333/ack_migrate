package utils

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetAllDeploymentAndStatefulSetWithLabels(kubeClient client.Client, matchLabels map[string]string) (*appsv1.DeploymentList, *appsv1.StatefulSetList, error) {
	deployments := &appsv1.DeploymentList{}
	if err := kubeClient.List(context.Background(), deployments, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(matchLabels),
	}); err != nil {
		klog.Errorf("Failed to list deployments: %v", err)
		return nil, nil, err
	}
	statefulSets := &appsv1.StatefulSetList{}
	if err := kubeClient.List(context.Background(), statefulSets, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(matchLabels),
	}); err != nil {
		klog.Errorf("Failed to list statefulSets: %v", err)
		return nil, nil, err
	}
	return deployments, statefulSets, nil
}
