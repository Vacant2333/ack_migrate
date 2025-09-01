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

func main() {
	var clusterID string
	flag.StringVar(&clusterID, "clusterid", "", "CloudPilot AI cluster id (required)")
	flag.Parse()

	if clusterID == "" {
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
	c := NewCloudPilotClient(ak, clusterID)

	// Require explicit "delete"
	if !requireExactInput("Type 'delete' to DELETE the current NodePools & NodeClasses on the server side, or anything else to skip: ", "delete") {
		klog.Infof("delete skipped by user; migration left original objects intact")
		return
	}
	// Delete from cluster
	if err := deleteAll(c); err != nil {
		fmt.Fprintf(os.Stderr, "error: delete failed: %v\n", err)
		os.Exit(2)
	}
	klog.Infof("delete finished successfully")

	var nodepoolList alibabacloudcorev1.NodePoolList
	if err := kubeClient.List(context.Background(), &nodepoolList); err != nil {
		panic(fmt.Errorf("failed to list nodepools: %v", err))
	}
	var nodeclassList alibabacloudproviderv1alpha1.ECSNodeClassList
	if err := kubeClient.List(context.Background(), &nodeclassList); err != nil {
		panic(fmt.Errorf("failed to list nodeclasses: %v", err))
	}

	// Preview tables
	printPreviewTables(nodepoolList.Items, nodeclassList.Items)

	// Require explicit "upload"
	if !requireExactInput("Type 'upload' to start uploading to CloudPilot AI, or anything else to abort: ", "upload") {
		klog.Infof("aborted by user; nothing uploaded, nothing deleted")
		return
	}

	// Upload to CloudPilot
	if err := uploadAll(c, nodeclassList.Items, nodepoolList.Items); err != nil {
		fmt.Fprintf(os.Stderr, "error: upload failed: %v\n", err)
		os.Exit(2)
	}
	klog.Infof("upload finished successfully")
}
