package utils

import (
	"context"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/cloudpilot-ai/lib/pkg/alibabacloud/karpenter/apis"
	apisv1 "github.com/cloudpilot-ai/lib/pkg/alibabacloud/karpenter/apis/v1"
	apisv1beta1 "github.com/cloudpilot-ai/lib/pkg/alibabacloud/karpenter/apis/v1beta1"
	"github.com/samber/lo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudpilot-ai/cloudpilot-agent/pkg/definitions"
	"github.com/cloudpilot-ai/cloudpilot-agent/pkg/values"
)

func DefaultBackoff(ctx context.Context) backoff.BackOffContext {
	return backoff.WithContext(backoff.WithMaxRetries(backoff.NewConstantBackOff(1*time.Second), 5), ctx)
}

func PatchNode(kubeClient client.Client, node *corev1.Node, changeFn func(*corev1.Node)) error {
	originalNode := node.DeepCopy()
	changeFn(node)

	ctx := context.Background()
	err := backoff.Retry(func() error {
		if reflect.DeepEqual(node, originalNode) {
			return nil
		}
		err := kubeClient.Patch(ctx, node, client.MergeFrom(originalNode))
		return err
	}, DefaultBackoff(ctx))
	if err != nil {
		klog.Errorf("Failed to patch node: %v", err)
		return err
	}

	return nil
}

func GetWorkloadPods(ctx context.Context, kubeClient client.Client, namespace string, selector *metav1.LabelSelector) (*corev1.PodList, error) {
	pods := &corev1.PodList{}

	if err := kubeClient.List(ctx, pods, &client.ListOptions{
		LabelSelector: labels.Set(selector.MatchLabels).AsSelector(),
		FieldSelector: nil,
		Namespace:     namespace,
	}); err != nil {
		klog.Errorf("Failed to list pods: %v", err)
		return nil, err
	}

	return pods, nil
}

func GetNodePods(ctx context.Context, kubeClient client.Client, all bool, nodes ...corev1.Node) ([]*corev1.Pod, error) {
	var pods []*corev1.Pod
	for _, node := range nodes {
		var podList corev1.PodList
		if err := kubeClient.List(ctx, &podList, client.MatchingFields{"spec.nodeName": node.Name}); err != nil {
			return nil, fmt.Errorf("failed to list pods %w", err)
		}
		for _, pod := range podList.Items {
			if all || ShouldReschedulePod(&pod) {
				pods = append(pods, pod.DeepCopy())
			}
		}
	}

	return pods, nil
}

func ShouldReschedulePod(targetPod *corev1.Pod) bool {
	// These pods don't need to be rescheduled.
	return !IsOwnedByNode(targetPod) &&
		!IsOwnedByDaemonSet(targetPod) &&
		!IsTerminal(targetPod) &&
		!IsTerminating(targetPod)
}

const (
	doNotDisruptLabelKey = apis.Group + "/do-not-disrupt"
)

func IsNodeRebalanceAble(ctx context.Context, kubeClient client.Client, node corev1.Node) (bool, error) {
	if node.Annotations != nil && node.Annotations[apisv1.DoNotDisruptAnnotationKey] == "true" {
		return false, nil
	}
	if node.Labels != nil && node.Labels[doNotDisruptLabelKey] == "true" {
		return false, nil
	}
	pods, err := GetNodePods(ctx, kubeClient, false, node)
	if err != nil {
		klog.Errorf("Failed to get node pods: %v", err)
		return false, err
	}

	for _, pod := range pods {
		if pod.Annotations[apisv1beta1.DoNotDisruptAnnotationKey] == "true" {
			return false, nil
		}
	}
	return true, nil
}

// IsNodeSpot return if the node is spot, ensure the node is non-nil.
func IsNodeSpot(node *corev1.Node, cloudProvider string) (bool, error) {
	capacityType, err := ExtractNodeCapacityType(cloudProvider, node)
	if err != nil {
		klog.Errorf("Failed to extract node capacity type: %v", err)
		return false, err
	}

	return capacityType == values.SpotCapacityType, nil
}

// IsPodSpot return if the pod is a spot pod, podNode can be nil if the node's name unset.
func IsPodSpot(pod *corev1.Pod, podNode *corev1.Node, cloudProvider string) (bool, error) {
	if podNode != nil {
		isSpot, err := IsNodeSpot(podNode, cloudProvider)
		if err != nil {
			klog.Errorf("Failed to determine spot node: %v", err)
			return false, err
		}
		return isSpot, nil
	}

	if pod.Annotations == nil {
		return false, nil
	}

	return pod.Annotations[corev1.PodDeletionCost] == "-1", nil
}

func ExtractNodeCapacityType(provider string, node *corev1.Node) (string, error) {
	if !lo.Contains(values.AllowedClusterProvider, provider) {
		return "", fmt.Errorf("provider %s is not supported", provider)
	}

	capacityType := node.Labels[apisv1beta1.CapacityTypeLabelKey]
	if capacityType == "" {
		switch provider {
		case values.CloudProviderAWS:
			capacityType = node.Labels[values.AWSNodeCapacityTypeLabelKey]
		case values.CloudProviderAlibabaCloud:
			// spot-strategy values: `NoSpot`, `SpotWithPriceLimit`, `SpotAsPriceGo`
			// `NoSpot` means `OnDemand`, others means `Spot` capacity type
			// Docs: https://help.aliyun.com/zh/ack/ack-managed-and-ack-dedicated/developer-reference/api-cs-2015-12-15-createclusternodepool
			spotStrategy := node.Labels["node.alibabacloud.com/spot-strategy"]
			// The label may not exist, we should keep the node capacity type to OnDemand
			if spotStrategy == "SpotWithPriceLimit" || spotStrategy == "SpotAsPriceGo" {
				capacityType = "spot"
			}
		}
	}

	// Default is ON_DEMAND
	capacityType = lo.Ternary(capacityType == "", "on-demand", capacityType)
	// Convert the on-demand to on_demand
	capacityType = lo.Ternary(capacityType == "on-demand", "on_demand", capacityType)

	// To unify the format: SPOT/ON_DEMAND
	return strings.ToUpper(capacityType), nil
}

// IsNodeRebalanced check if the node is managed by CloudPilot AI
func IsNodeRebalanced(nodeLabels map[string]string) bool {
	if nodeLabels == nil {
		return false
	}

	return nodeLabels[values.CloudPilotManagedNodeLabelKey] == "true"
}

func ExtractNodeZoneName(node *corev1.Node) string {
	if node.Labels == nil {
		return ""
	}
	return node.Labels[corev1.LabelTopologyZone]
}

func ExtractAWSNodeZoneID(node *corev1.Node) string {
	if node.Labels == nil {
		return ""
	}
	return node.Labels["topology.k8s.aws/zone-id"]
}

func ExtractInstanceFamily(instanceType string) string {
	if instanceType == "" {
		return ""
	}
	parts := strings.Split(instanceType, ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func ExtractNodeInstanceType(node *corev1.Node) string {
	return node.Labels[corev1.LabelInstanceType]
}

func ExtractPodOwnerWorkload(c client.Client, pod *corev1.Pod) (string, string, string, bool) {
	if pod.OwnerReferences == nil {
		return "", "", "", false
	}
	// We don't find what other workload type can create ReplicaSet except Deployment.
	switch pod.OwnerReferences[0].Kind {
	case "ReplicaSet":
		deploymentName, deploymentNs, err := ExtractDeploymentFromReplicaSet(c, pod.Namespace, pod.OwnerReferences[0].Name)
		if err != nil {
			klog.Errorf("Failed to extract deployment from ReplicaSet %s/%s: %v",
				pod.Namespace, pod.OwnerReferences[0].Name, err)
			return "", "", "", false
		}

		return definitions.DeploymentWorkloadType, deploymentNs, deploymentName, true
	case "StatefulSet":
		return definitions.StatefulSetWorkloadType, pod.Namespace, pod.OwnerReferences[0].Name, true
	default:
		return "", "", "", false
	}
}

func ExtractDeploymentFromReplicaSet(c client.Client, rsNamespace, rsName string) (string, string, error) {
	var rs appsv1.ReplicaSet
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: rsNamespace,
		Name:      rsName,
	}, &rs); err != nil {
		return "", "", fmt.Errorf("failed to get ReplicaSet %s/%s: %v", rsNamespace, rsName, err)
	}

	for _, owner := range rs.OwnerReferences {
		if owner.Kind == "Deployment" && owner.Controller != nil && *owner.Controller {
			return owner.Name, rsNamespace, nil
		}
	}

	return "", "", fmt.Errorf("replicaset %s/%s has no deployment controller", rsNamespace, rsName)
}

func ExtractWorkloadConfigFromLabels(labels map[string]string) (bool, int32) {
	if labels == nil {
		return true, 0
	}

	spotFriendly, rok := labels[values.CloudPilotSpotFriendlyLabelKey]
	if rok && spotFriendly == values.CloudPilotSpotFriendlyLabelValueFalse {
		return false, 0
	}

	minNonSpotReplicasValue, mok := labels[values.CloudPilotMinNonSpotReplicasLabelKey]
	if !mok {
		return true, 0
	}

	minNonSpotReplicas, err := strconv.ParseInt(minNonSpotReplicasValue, 10, 64)
	if err != nil {
		klog.Errorf("Failed to parse %d: %v", minNonSpotReplicas, err)
		return true, 0
	}

	return true, int32(minNonSpotReplicas)
}

func ExtractWorkloadRebalanceAbleFromAnnotations(annotations map[string]string) bool {
	if annotations == nil {
		return true
	}
	return annotations[apisv1beta1.DoNotDisruptAnnotationKey] != "true"
}

func CopyMap[K comparable, V any](original map[K]V) map[K]V {
	copied := make(map[K]V)
	for key, value := range original {
		copied[key] = value
	}
	return copied
}

func RoundFloat(value float64, decimalPlaces int) float64 {
	factor := math.Pow(10, float64(decimalPlaces))
	return math.Round(value*factor) / factor
}

func HasBoundPVC(pod *corev1.Pod) bool {
	for _, v := range pod.Spec.Volumes {
		// TODO: only block storage should be considered.
		if v.PersistentVolumeClaim != nil {
			return true
		}
	}
	return false
}

const maxDepth = 10

func FindRootOwner(ctx context.Context, kubeClient client.Client, namespace string, ownerRef metav1.OwnerReference) (unstructured.Unstructured, error) {
	obj := unstructured.Unstructured{}

	for range maxDepth {
		obj.SetGroupVersionKind(schema.FromAPIVersionAndKind(ownerRef.APIVersion, ownerRef.Kind))
		if err := kubeClient.Get(ctx, client.ObjectKey{
			Namespace: namespace,
			Name:      ownerRef.Name,
		}, &obj); err != nil {
			return unstructured.Unstructured{}, err
		}

		owners := obj.GetOwnerReferences()
		if len(owners) == 0 {
			return obj, nil
		}
		ownerRef = owners[0]
	}

	return unstructured.Unstructured{}, fmt.Errorf("%v reached maximum recursion depth %d without finding ultimate owner", ownerRef.Name, maxDepth)
}

func GetEnvDefault(env string, defaultValue string) string {
	value := os.Getenv(env)
	if value == "" {
		return defaultValue
	}
	return value
}

// IsOwnedByNode returns true if the pod is a static pod owned by a specific node
func IsOwnedByNode(pod *corev1.Pod) bool {
	return IsOwnedBy(pod, []schema.GroupVersionKind{
		{Version: "v1", Kind: "Node"},
	})
}

func IsOwnedBy(pod *corev1.Pod, gvks []schema.GroupVersionKind) bool {
	for _, ignoredOwner := range gvks {
		for _, owner := range pod.ObjectMeta.OwnerReferences {
			if owner.APIVersion == ignoredOwner.GroupVersion().String() && owner.Kind == ignoredOwner.Kind {
				return true
			}
		}
	}
	return false
}

func IsOwnedByDaemonSet(pod *corev1.Pod) bool {
	return IsOwnedBy(pod, []schema.GroupVersionKind{
		{Group: "apps", Version: "v1", Kind: "DaemonSet"},
	})
}

func IsTerminal(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded
}

func IsTerminating(pod *corev1.Pod) bool {
	return pod.DeletionTimestamp != nil
}

func GetKubeConfig(debugCluster string) (*rest.Config, error) {
	if debugCluster != "" {
		config, err := clientcmd.BuildConfigFromFlags("", debugCluster)
		return config, err
	}

	config, err := rest.InClusterConfig()
	return config, err
}
