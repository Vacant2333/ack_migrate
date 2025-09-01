package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	alibabacloudproviderv1alpha1 "github.com/cloudpilot-ai/lib/pkg/alibabacloud/karpenter-provider-alibabacloud/apis/v1alpha1"
	alibabacloudcorev1 "github.com/cloudpilot-ai/lib/pkg/alibabacloud/karpenter/apis/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	_ = alibabacloudproviderv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme)
	_ = alibabacloudcorev1.SchemeBuilder.AddToScheme(scheme.Scheme)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage:
  ack_migrate --clusterid <id>

Flags:
`)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Examples:
  ack_migrate --clusterid 9b71a8d5-1500-5e64-957d-5fa75a1b0cb2
`)
}

func main() {
	var clusterID string
	flag.StringVar(&clusterID, "clusterid", "", "CloudPilot AI cluster id (required)")
	flag.Usage = printUsage
	flag.Parse()

	if clusterID == "" {
		flag.Usage()
		panic("--clusterid is required")
	}
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		panic(fmt.Errorf("KUBECONFIG env is empty"))
	}
	ak := os.Getenv("CLOUDPILOT_API_KEY")
	if ak == "" {
		panic(fmt.Errorf("CLOUDPILOT_API_KEY env is empty"))
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(fmt.Errorf("failed to create config: %v", err))
	}

	kubeClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		panic(fmt.Errorf("failed to create client: %v", err))
	}

	var nodepoolList alibabacloudcorev1.NodePoolList
	if err := kubeClient.List(context.Background(), &nodepoolList); err != nil {
		panic(fmt.Errorf("failed to list nodepools: %v", err))
	}
	for _, nodepool := range nodepoolList.Items {
		klog.Infof("nodepool: %s", nodepool.Name)
	}

	var nodeclassList alibabacloudproviderv1alpha1.ECSNodeClassList
	if err := kubeClient.List(context.Background(), &nodeclassList); err != nil {
		panic(fmt.Errorf("failed to list nodeclasses: %v", err))
	}
	for _, nodeclass := range nodeclassList.Items {
		klog.Infof("nodeclass: %s", nodeclass.Name)
	}

	c := NewCloudPilotClient(ak, clusterID)
	for _, nc := range nodeclassList.Items {
		klog.Infof("migrating nodeclass: %s", nc.Name)
		err := c.ApplyNodeClass(RebalanceNodeClass{ECSNodeClass: &ECSNodeClass{
			Name:          nc.Name,
			NodeClassSpec: &nc.Spec,
		}})
		if err != nil {
			panic(fmt.Errorf("failed to migrate nodeclass %s to CloudPilot AI: %v", nc.Name, err))
		}
	}

	for _, np := range nodepoolList.Items {
		klog.Infof("migrating nodepool: %s", np.Name)
		err := c.ApplyNodePool(RebalanceNodePool{ECSNodePool: &ECSNodePool{
			Name:         np.Name,
			Enable:       false,
			NodePoolSpec: &np.Spec,
		}})
		if err != nil {
			panic(fmt.Errorf("failed to migrate nodepool %s to CloudPilot AI: %v", np.Name, err))
		}
	}

	klog.Infof("Migrate completed")
}
