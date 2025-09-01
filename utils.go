package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	alibabacloudproviderv1alpha1 "github.com/cloudpilot-ai/lib/pkg/alibabacloud/karpenter-provider-alibabacloud/apis/v1alpha1"
	alibabacloudcorev1 "github.com/cloudpilot-ai/lib/pkg/alibabacloud/karpenter/apis/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// deleteAll removes the exact listed NodeClasses and NodePools from the cluster.
// Uses foreground propagation to ensure cascading; adjust if your CRDs need background.
func deleteAll(
	ctx context.Context,
	kube client.Client,
	nodeclasses []alibabacloudproviderv1alpha1.ECSNodeClass,
	nodepools []alibabacloudcorev1.NodePool,
) error {
	// Delete NodePools first if they reference NodeClasses; adjust order if needed.
	policy := metav1.DeletePropagationForeground
	opts := &client.DeleteOptions{PropagationPolicy: &policy}

	// Delete NodePools
	for i := range nodepools {
		np := &nodepools[i] // IMPORTANT: use index to avoid taking address of range variable
		klog.Infof("deleting nodepool: %s", np.Name)
		if err := kube.Delete(ctx, np, opts); err != nil {
			return fmt.Errorf("delete nodepool %q: %w", np.Name, err)
		}
	}

	// Delete NodeClasses
	for i := range nodeclasses {
		nc := &nodeclasses[i]
		klog.Infof("deleting nodeclass: %s", nc.Name)
		if err := kube.Delete(ctx, nc, opts); err != nil {
			return fmt.Errorf("delete nodeclass %q: %w", nc.Name, err)
		}
	}
	return nil
}

func uploadAll(
	ctx context.Context,
	c *Client, // TODO: adjust to your real client type
	nodeclasses []alibabacloudproviderv1alpha1.ECSNodeClass,
	nodepools []alibabacloudcorev1.NodePool,
) error {
	// Upload NodeClasses
	for i := range nodeclasses {
		nc := &nodeclasses[i]
		klog.Infof("uploading nodeclass: %s", nc.Name)
		if err := c.ApplyNodeClass(RebalanceNodeClass{
			ECSNodeClass: &ECSNodeClass{
				Name:          nc.Name,
				NodeClassSpec: &nc.Spec,
			},
		}); err != nil {
			return fmt.Errorf("apply nodeclass %q: %w", nc.Name, err)
		}
	}

	// Upload NodePools
	for i := range nodepools {
		np := &nodepools[i]
		klog.Infof("uploading nodepool: %s", np.Name)
		if err := c.ApplyNodePool(RebalanceNodePool{
			ECSNodePool: &ECSNodePool{
				Name:         np.Name,
				Enable:       false, // keep disabled initially; adjust to your policy
				NodePoolSpec: &np.Spec,
			},
		}); err != nil {
			return fmt.Errorf("apply nodepool %q: %w", np.Name, err)
		}
	}
	return nil
}

// ---- Preview table & helpers ----

func printPreviewTables(
	nodepools []alibabacloudcorev1.NodePool,
	nodeclasses []alibabacloudproviderv1alpha1.ECSNodeClass,
) {
	// Stable sort
	sort.Slice(nodepools, func(i, j int) bool { return nodepools[i].Name < nodepools[j].Name })
	sort.Slice(nodeclasses, func(i, j int) bool { return nodeclasses[i].Name < nodeclasses[j].Name })

	// NodePools
	fmt.Println("\n=== NodePools Preview ===")
	npw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(npw, "NAME\tSPEC (truncated)")
	for _, np := range nodepools {
		fmt.Fprintf(npw, "%s\t%s\n",
			np.Name,
			trim(compactJSON(np.Spec), 120),
		)
	}
	npw.Flush()

	// NodeClasses
	fmt.Println("\n=== ECSNodeClasses Preview ===")
	ncw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(ncw, "NAME\tSPEC (truncated)")
	for _, nc := range nodeclasses {
		fmt.Fprintf(ncw, "%s\t%s\n",
			nc.Name,
			trim(compactJSON(nc.Spec), 120),
		)
	}
	ncw.Flush()

	fmt.Printf("\nSummary: %d NodePool(s), %d NodeClass(es)\n", len(nodepools), len(nodeclasses))
}

func compactJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("<marshal error: %v>", err)
	}
	return string(b)
}

func trim(s string, max int) string {
	if max <= 3 || len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// requireExactInput prompts and returns true only if the exact expected (case-insensitive) token is entered.
func requireExactInput(prompt, expected string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	line, _ := reader.ReadString('\n')
	resp := strings.ToLower(strings.TrimSpace(line))
	return resp == strings.ToLower(expected)
}
