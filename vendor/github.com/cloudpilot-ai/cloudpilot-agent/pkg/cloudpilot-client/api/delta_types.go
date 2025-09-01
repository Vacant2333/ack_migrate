package api

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	karpv1 "github.com/cloudpilot-ai/lib/pkg/alibabacloud/karpenter/apis/v1"
	"github.com/samber/lo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
	metricsv1beta1api "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudpilot-ai/cloudpilot-agent/pkg/utils"
)

type StateDelta struct {
	NodeDeltaMutex     sync.Mutex               `json:"-"`
	NodeDeltas         map[string]NodeDelta     `json:"nodeDeltas"`
	WorkloadDeltaMutex sync.Mutex               `json:"-"`
	WorkloadDeltas     map[string]WorkloadDelta `json:"workloadDeltas"`

	UpdatedAfterRebalance bool `json:"updatedAfterRebalance"`

	ClusterUsedResource        corev1.ResourceList `json:"clusterUsedResource"`
	ClusterRequestResource     corev1.ResourceList `json:"clusterRequestResource"`
	ClusterAllocatableResource corev1.ResourceList `json:"clusterAllocatableResource"`

	NamespacesKindsWorkloadsResources NamespacesKindsWorkloadsResources `json:"namespacesKindsWorkloadsResources"`

	ClusterAllocatedResourceRate ResourceRate `json:"clusterAllocatedResourceRate"`
}

type (
	WorkloadsResources                map[string]WorkloadResource        // key is the uid of the workload
	KindsWorkloadsResources           map[string]WorkloadsResources      // key is groupVersionKind
	NamespacesKindsWorkloadsResources map[string]KindsWorkloadsResources // key is the uid of the namespace
)

type WorkloadResource struct {
	// Name when disableWorkloadUploading is false, the name and namespace may be empty
	Name      string `json:"name"`
	Namespace string `json:"namespace"`

	Replicas      int32 `json:"replicas"`
	ReadyReplicas int32 `json:"readyReplicas"`

	UsedResources    corev1.ResourceList `json:"usedResource"`
	RequestResources corev1.ResourceList `json:"requestResource"`
}

type NodeDelta struct {
	ID                       string      `json:"id"`
	Event                    EventType   `json:"event"`
	InstanceType             string      `json:"instanceType"`
	CapacityType             string      `json:"capacityType"`
	IPAddress                []string    `json:"ipAddress"`
	Zone                     string      `json:"zone"`
	AWSZoneID                string      `json:"AWSZoneID"`
	Rebalanced               *bool       `json:"rebalanced"`
	Status                   string      `json:"status"`
	Message                  string      `json:"message"`
	StatusLastTransitionTime metav1.Time `json:"statusLastTransitionTime"`

	UsedResources        corev1.ResourceList `json:"usedResources"`
	RequestResources     corev1.ResourceList `json:"requestResources"`
	AllocatableResources corev1.ResourceList `json:"allocatableResources"`
	ProvisionedResources corev1.ResourceList `json:"provisionedResources"`

	ReadyPodNumber    int `json:"readyPodNumber"`
	NotReadyPodNumber int `json:"notReadyPodNumber"`

	Owner string `json:"owner"`

	CreationTimestamp int64 `json:"creationTimestamp"`
	DeletionTimestamp int64 `json:"deletionTimestamp"`
}

type WorkloadDelta struct {
	Event        EventType `json:"event"`
	WorkloadType string    `json:"workloadType"`
	Namespace    string    `json:"namespace"`
	Replicas     int       `json:"replicas"`
}

type EventType string

const (
	EventTypeAdd            EventType = "add"
	EventTypeUpdate         EventType = "update"
	EventTypeDelete         EventType = "delete"
	DeploymentWorkloadType            = "deployment"
	StatefulSetWorkloadType           = "statefulset"
	// TBD: We don't know how to handle the Job type now.
)

func (s *StateDelta) clone() *StateDelta {
	clone := &StateDelta{
		NodeDeltas:                        map[string]NodeDelta{},
		WorkloadDeltas:                    map[string]WorkloadDelta{},
		UpdatedAfterRebalance:             s.UpdatedAfterRebalance,
		ClusterUsedResource:               s.ClusterUsedResource.DeepCopy(),
		ClusterRequestResource:            s.ClusterRequestResource.DeepCopy(),
		ClusterAllocatableResource:        s.ClusterAllocatableResource.DeepCopy(),
		NamespacesKindsWorkloadsResources: NamespacesKindsWorkloadsResources{},
	}
	for k, v := range s.NodeDeltas {
		clone.NodeDeltas[k] = v
	}
	for k, v := range s.WorkloadDeltas {
		clone.WorkloadDeltas[k] = v
	}
	for k, v := range s.NamespacesKindsWorkloadsResources {
		clone.NamespacesKindsWorkloadsResources[k] = v
	}
	return clone
}

func (s *StateDelta) AddDeltasIfNotExist(delta *StateDelta) {
	if delta == nil {
		klog.Warning("Received nil delta in AddDeltasIfNotExist")
		return
	}

	s.NodeDeltaMutex.Lock()
	s.WorkloadDeltaMutex.Lock()

	defer s.NodeDeltaMutex.Unlock()
	defer s.WorkloadDeltaMutex.Unlock()

	for nodeName, nodeData := range delta.NodeDeltas {
		if _, ok := s.NodeDeltas[nodeName]; !ok {
			s.NodeDeltas[nodeName] = nodeData
		}
	}

	for key, workloadData := range delta.WorkloadDeltas {
		if _, ok := s.WorkloadDeltas[key]; !ok {
			s.WorkloadDeltas[key] = workloadData
		}
	}
}

func (s *StateDelta) AddWorkloadDeltasIfNotExist(delta *StateDelta) {
	s.WorkloadDeltaMutex.Lock()
	defer s.WorkloadDeltaMutex.Unlock()

	for key, workloadData := range delta.WorkloadDeltas {
		if _, ok := s.WorkloadDeltas[key]; !ok {
			s.WorkloadDeltas[key] = workloadData
		}
	}
}

func (s *StateDelta) AddWorkloadDelta(workloadName string, delta WorkloadDelta) {
	s.WorkloadDeltaMutex.Lock()
	defer s.WorkloadDeltaMutex.Unlock()

	key := fmt.Sprintf("%s/%s/%s", delta.WorkloadType, delta.Namespace, workloadName)
	s.WorkloadDeltas[key] = delta
}

func (s *StateDelta) AddNodeDelta(name string, delta NodeDelta) {
	s.NodeDeltaMutex.Lock()
	defer s.NodeDeltaMutex.Unlock()

	s.NodeDeltas[name] = delta
}

func (s *StateDelta) EncodeForSending(ctx context.Context, kubeClient client.Client,
	metricClient metricsclientset.Interface, isDeltas bool, disableWorkloadUploading bool) ([]byte, *StateDelta, error) {
	s.NodeDeltaMutex.Lock()
	s.WorkloadDeltaMutex.Lock()

	defer s.NodeDeltaMutex.Unlock()
	defer s.WorkloadDeltaMutex.Unlock()

	var nodes corev1.NodeList
	if err := kubeClient.List(ctx, &nodes, &client.ListOptions{}); err != nil {
		return nil, nil, err
	}

	if isDeltas && len(s.NodeDeltas) == 0 && len(s.WorkloadDeltas) == 0 {
		return nil, nil, nil
	}

	s.UpdatedAfterRebalance = !containsNonManagedNodes(ctx, kubeClient, nodes.Items)
	s.ClusterAllocatableResource = GetClusterAllocatableResources(nodes.Items)

	var err error
	s.ClusterUsedResource, err = GetClusterUsedResources(ctx, metricClient, metav1.ListOptions{})
	if err != nil {
		klog.Warningf("failed to get cluster used resources: %v", err)
	}

	s.NamespacesKindsWorkloadsResources, s.ClusterRequestResource, err = GetNamespacesKindsWorkloadsResourcesAndRequestResources(ctx,
		kubeClient, metricClient, disableWorkloadUploading)
	if err != nil {
		return nil, nil, err
	}

	s.ClusterAllocatedResourceRate, err = CalculateResourceRate(s.ClusterAllocatableResource, s.ClusterRequestResource)
	if err != nil {
		klog.Errorf("failed to calculate cluster allocated resource rate: %v", err)
		return nil, nil, err
	}

	nodeNameMap := lo.SliceToMap(nodes.Items, func(node v1.Node) (string, *v1.Node) {
		return node.Name, &node
	})

	for name, nodeDelta := range s.NodeDeltas {
		node, ok := nodeNameMap[name]
		if !ok || node == nil {
			continue
		}

		if err := PopulateNodeDelta(ctx, &nodeDelta, node, metricClient, kubeClient); err != nil {
			klog.Errorf("failed to populate node delta: %v", err)
			continue
		}

		s.NodeDeltas[name] = nodeDelta
	}

	encodeData, err := json.Marshal(s)
	if err != nil {
		klog.Errorf("failed to encode state delta: %v", err)
		return nil, nil, err
	}

	clone := s.clone()
	s.NodeDeltas = map[string]NodeDelta{}
	s.WorkloadDeltas = map[string]WorkloadDelta{}
	return encodeData, clone, nil
}

func PopulateNodeDelta(ctx context.Context, nodeDelta *NodeDelta, node *v1.Node, metricClient metricsclientset.Interface, kubeClient client.Client) error {
	if nodeDelta == nil || node == nil {
		return nil
	}

	var err error
	nodeDelta.IPAddress = nodeIPAddress(node)
	nodeDelta.UsedResources, err = nodeUsedResources(node, metricClient)
	if err != nil {
		klog.Warningf("failed to get node used resources: %v", err)
	}

	nodePods, err := utils.GetNodePods(ctx, kubeClient, true, *node)
	if err != nil {
		return err
	}

	nodeDelta.RequestResources = nodeRequestResources(nodePods)
	nodeDelta.AllocatableResources = nodeAllocatableResources(node)
	nodeDelta.ProvisionedResources = nodeProvisionedResources(node)

	nodeDelta.ReadyPodNumber = 0
	nodeDelta.NotReadyPodNumber = 0
	for pi := range nodePods {
		if utils.CheckPodStatus(nodePods[pi]) {
			nodeDelta.ReadyPodNumber++
			continue
		}
		nodeDelta.NotReadyPodNumber++
	}

	return nil
}

func GetNamespacesKindsWorkloadsResourcesAndRequestResources(ctx context.Context, kubeClient client.Client,
	metricClient metricsclientset.Interface, disableWorkloadUploading bool) (
	NamespacesKindsWorkloadsResources, corev1.ResourceList, error) {
	podList := &corev1.PodList{}
	if err := kubeClient.List(ctx, podList); err != nil {
		return nil, nil, err
	}

	var (
		namespacesWorkloadResources = make(NamespacesKindsWorkloadsResources, len(podList.Items))
		clusterRequestResources     = corev1.ResourceList{}

		owner unstructured.Unstructured
	)

	podMetrics := map[string]corev1.ResourceList{}
	metrics, err := getPodMetricsFromMetricsAPI(ctx, metricClient)
	if err != nil {
		klog.Warningf("failed to get pod metrics from metrics API: %v", err)
	} else {
		podMetrics = make(map[string]corev1.ResourceList, len(metrics.Items))
		for mi := range metrics.Items {
			podUsedResources := sumContainerMetricsUsedResources(metrics.Items[mi].Containers)
			podMetrics[metrics.Items[mi].Name] = podUsedResources
		}
	}

	for pi := range podList.Items {
		podRequestResources := corev1.ResourceList{}
		if podList.Items[pi].Status.Phase == corev1.PodRunning || podList.Items[pi].Status.Phase == corev1.PodPending {
			podRequestResources = utils.SumResourceRequests(podList.Items[pi].Spec.Containers)
			utils.AddResourceListsInPlace(clusterRequestResources, podRequestResources)
		}

		resolveWorkloadResourceWithOwner(ctx, &podList.Items[pi], owner, podRequestResources, podMetrics,
			namespacesWorkloadResources, kubeClient, disableWorkloadUploading)
	}

	return namespacesWorkloadResources, clusterRequestResources, nil
}

func resolveWorkloadResourceWithOwner(
	ctx context.Context,
	pod *corev1.Pod,
	owner unstructured.Unstructured,
	podRequestResources corev1.ResourceList,
	podMetrics map[string]corev1.ResourceList,
	namespacesWorkloadResources NamespacesKindsWorkloadsResources,
	kubeClient client.Client,
	disableWorkloadUploading bool,
) {
	if len(pod.OwnerReferences) == 0 {
		// TODO: Pods without Owner Reference are not counted
		return
	}
	var ns corev1.Namespace
	if err := kubeClient.Get(ctx, types.NamespacedName{Name: pod.Namespace}, &ns); err != nil {
		klog.Errorf("failed to get namespace %s, err: %v", pod.Namespace, err)
		return
	}

	owner, err := utils.FindRootOwner(ctx, kubeClient, pod.Namespace, pod.OwnerReferences[0])
	if err != nil {
		klog.Error(err)
		return
	}

	kindsWorkloadsResources, ok := namespacesWorkloadResources[string(ns.UID)]
	if !ok {
		kindsWorkloadsResources = make(KindsWorkloadsResources)
	}

	workloadsResources, ok := kindsWorkloadsResources[owner.GroupVersionKind().String()]
	if !ok {
		workloadsResources = make(WorkloadsResources)
	}

	workloadResource, ok := workloadsResources[string(owner.GetUID())]
	if !ok {
		workloadResource = WorkloadResource{
			UsedResources:    corev1.ResourceList{},
			RequestResources: corev1.ResourceList{},
		}
	}
	if !disableWorkloadUploading {
		workloadResource.Name = owner.GetName()
		workloadResource.Namespace = owner.GetNamespace()
	}

	// Add ReadyReplicas
	if utils.CheckPodStatus(pod) {
		workloadResource.ReadyReplicas++
	}

	// Add Replicas
	if workloadResource.Replicas == 0 {
		workloadResource.Replicas, err = GetWorkloadReplicas(ctx, kubeClient, owner.GetNamespace(), owner.GetName(), owner.GroupVersionKind())
		if err != nil {
			klog.Error(err)
		}
	}

	// Add RequestResources
	if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending {
		utils.AddResourceListsInPlace(workloadResource.RequestResources, podRequestResources)
	}

	// Add UsedResources
	if podResource, ok := podMetrics[pod.Name]; ok {
		utils.AddResourceListsInPlace(workloadResource.UsedResources, podResource)
	}

	workloadsResources[string(owner.GetUID())] = workloadResource
	kindsWorkloadsResources[owner.GroupVersionKind().String()] = workloadsResources
	namespacesWorkloadResources[string(ns.UID)] = kindsWorkloadsResources
}

func sumContainerMetricsUsedResources(containers []metricsapi.ContainerMetrics) corev1.ResourceList {
	var cpuUsage, memoryUsage resource.Quantity
	for ci := range containers {
		cpuUsage.Add(*containers[ci].Usage.Cpu())
		memoryUsage.Add(*containers[ci].Usage.Memory())
	}

	return corev1.ResourceList{
		corev1.ResourceCPU:    cpuUsage,
		corev1.ResourceMemory: memoryUsage,
	}
}

var (
	DeploymentGVK = schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}

	StatefulSetGVK = schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "StatefulSet",
	}

	DaemonSetGVK = schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "DaemonSet",
	}
)

func GetWorkloadReplicas(ctx context.Context, kubeClient client.Client, namespace, workloadName string, gvk schema.GroupVersionKind) (int32, error) {
	switch gvk {
	case DeploymentGVK:
		var deployment appsv1.Deployment
		if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: workloadName}, &deployment); err != nil {
			return 0, err
		}
		return *deployment.Spec.Replicas, nil
	case StatefulSetGVK:
		var statefulSet appsv1.StatefulSet
		if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: workloadName}, &statefulSet); err != nil {
			return 0, err
		}
		return *statefulSet.Spec.Replicas, nil
	case DaemonSetGVK:
		var daemonSet appsv1.DaemonSet
		if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: workloadName}, &daemonSet); err != nil {
			return 0, err
		}
		return daemonSet.Status.DesiredNumberScheduled, nil
	default:
		// TODO: support more workload
		return 0, fmt.Errorf("unsupported workload type: %s", gvk.String())
	}
}

func GetClusterUsedResources(ctx context.Context, metricClient metricsclientset.Interface, options metav1.ListOptions) (corev1.ResourceList, error) {
	metrics, err := getNodeMetricsFromMetricsAPI(ctx, metricClient, options)
	if err != nil {
		return corev1.ResourceList{}, err
	}

	var cpuUsage, memoryUsage resource.Quantity
	for mi := range metrics.Items {
		cpuUsage.Add(*metrics.Items[mi].Usage.Cpu())
		memoryUsage.Add(*metrics.Items[mi].Usage.Memory())
	}

	return corev1.ResourceList{
		corev1.ResourceCPU:    cpuUsage,
		corev1.ResourceMemory: memoryUsage,
	}, nil
}

func GetClusterAllocatableResources(nodes []corev1.Node) corev1.ResourceList {
	clusterAllocatableResources := corev1.ResourceList{}
	for ni := range nodes {
		utils.AddResourceListsInPlace(clusterAllocatableResources, nodes[ni].Status.Allocatable)
	}
	return clusterAllocatableResources
}

func containsNonManagedNodes(ctx context.Context, kubeClient client.Client, nodes []corev1.Node) bool {
	for _, n := range nodes {
		if n.Labels == nil {
			return true
		}
		ok, err := utils.IsNodeRebalanceAble(ctx, kubeClient, n)
		if err != nil || !ok {
			continue
		}
		_, ok = n.Labels[karpv1.NodePoolLabelKey]
		if !ok {
			return true
		}
	}
	return false
}

func getNodeMetricsFromMetricsAPI(ctx context.Context, metricsClient metricsclientset.Interface, options metav1.ListOptions) (*metricsapi.NodeMetricsList, error) {
	versionedMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, options)
	if err != nil {
		return nil, err
	}

	metrics := &metricsapi.NodeMetricsList{}
	if err := metricsv1beta1api.Convert_v1beta1_NodeMetricsList_To_metrics_NodeMetricsList(versionedMetrics, metrics, nil); err != nil {
		return nil, err
	}
	return metrics, nil
}

func getPodMetricsFromMetricsAPI(ctx context.Context, metricsClient metricsclientset.Interface) (*metricsapi.PodMetricsList, error) {
	versionedMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	metrics := &metricsapi.PodMetricsList{}
	err = metricsv1beta1api.Convert_v1beta1_PodMetricsList_To_metrics_PodMetricsList(versionedMetrics, metrics, nil)
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

func nodeIPAddress(node *v1.Node) []string {
	if node == nil || node.Status.Addresses == nil {
		return []string{}
	}

	ipAddresses := []string{}
	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeInternalIP {
			ipAddresses = append(ipAddresses, address.Address)
		}
	}

	return ipAddresses
}

func nodeUsedResources(node *v1.Node, metricClient metricsclientset.Interface) (corev1.ResourceList, error) {
	if node == nil {
		return corev1.ResourceList{}, nil
	}

	metrics, err := GetClusterUsedResources(context.Background(), metricClient, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", node.Name),
	})
	if err != nil {
		return corev1.ResourceList{}, err
	}

	return metrics, nil
}

func nodeRequestResources(nodePods []*corev1.Pod) corev1.ResourceList {
	requestResources := corev1.ResourceList{}
	for pi := range nodePods {
		if nodePods[pi] == nil {
			continue
		}
		if nodePods[pi].Status.Phase == corev1.PodRunning || nodePods[pi].Status.Phase == corev1.PodPending {
			utils.AddResourceListsInPlace(requestResources, utils.SumResourceRequests(nodePods[pi].Spec.Containers))
		}
	}

	return requestResources
}

func nodeAllocatableResources(node *v1.Node) corev1.ResourceList {
	if node == nil || node.Status.Allocatable == nil {
		return corev1.ResourceList{}
	}

	return node.Status.Allocatable
}

func nodeProvisionedResources(node *v1.Node) corev1.ResourceList {
	if node == nil || node.Status.Capacity == nil {
		return corev1.ResourceList{}
	}

	return node.Status.Capacity
}
