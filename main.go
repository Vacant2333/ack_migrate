package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	cloudpilotclient "github.com/cloudpilot-ai/cloudpilot-agent/pkg/cloudpilot-client"
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
  ack_migrate --clustername <name>

Flags:
`)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Examples:
  ack_migrate --clustername prod-cluster
`)
}

func main() {
	var clusterName string
	flag.StringVar(&clusterName, "clustername", "", "Kubernetes cluster name (required)")
	flag.Usage = printUsage
	flag.Parse()

	if clusterName == "" {
		flag.Usage()
		panic("--clustername is required")
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
		panic(fmt.Errorf("failed to create config:%v", err))
	}

	kubeClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		panic(fmt.Errorf("failed to create client:%v", err))
	}

	var nodepoolList alibabacloudcorev1.NodePoolList
	if err := kubeClient.List(context.Background(), &nodepoolList); err != nil {
		panic(fmt.Errorf("failed to list nodepools:%v", err))
	}
	for _, nodepool := range nodepoolList.Items {
		klog.Infof("nodepool: %s", nodepool.Name)
	}

	var nodeclassList alibabacloudproviderv1alpha1.ECSNodeClassList
	if err := kubeClient.List(context.Background(), &nodeclassList); err != nil {
		panic(fmt.Errorf("failed to list nodeclasses:%v", err))
	}
	for _, nodeclass := range nodeclassList.Items {
		klog.Infof("nodeclass: %s", nodeclass.Name)
	}
}
