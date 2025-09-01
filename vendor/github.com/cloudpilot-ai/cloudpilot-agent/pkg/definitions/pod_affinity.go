package definitions

import (
	"github.com/cloudpilot-ai/lib/pkg/alibabacloud/karpenter/apis/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	"github.com/cloudpilot-ai/cloudpilot-agent/pkg/values"
)

var karpenterPreferSpotNodeTerm = corev1.PreferredSchedulingTerm{
	Preference: corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{
			{
				Key:      v1beta1.CapacityTypeLabelKey,
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{v1beta1.CapacityTypeSpot},
			},
		},
	},
	Weight: 1,
}

var awsPreferSpotNodeTerm = corev1.PreferredSchedulingTerm{
	Preference: corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{
			{
				Key:      values.AWSNodeCapacityTypeLabelKey,
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{values.SpotCapacityType},
			},
		},
	},
	Weight: 1,
}

var karpenterRequireNotInSpotNodeTerm = corev1.NodeSelectorRequirement{
	Key:      v1beta1.CapacityTypeLabelKey,
	Operator: corev1.NodeSelectorOpNotIn,
	Values:   []string{v1beta1.CapacityTypeSpot},
}

var awsRequireNotInSpotNodeTerm = corev1.NodeSelectorRequirement{
	Key:      values.AWSNodeCapacityTypeLabelKey,
	Operator: corev1.NodeSelectorOpNotIn,
	Values:   []string{values.SpotCapacityType},
}

// GetPreferSpotNodeTerms Use this method to get Terms to prevent accidental modification of constants.
func GetPreferSpotNodeTerms(cloudProvider string) []corev1.PreferredSchedulingTerm {
	switch cloudProvider {
	case values.CloudProviderAWS:
		return []corev1.PreferredSchedulingTerm{
			karpenterPreferSpotNodeTerm,
			awsPreferSpotNodeTerm,
		}
	case values.CloudProviderAlibabaCloud:
		return []corev1.PreferredSchedulingTerm{
			karpenterPreferSpotNodeTerm,
		}
	default:
		klog.Errorf("Unsupported cloud provider: %s, when get prefer spot node terms", cloudProvider)
		return []corev1.PreferredSchedulingTerm{}
	}
}

// GetRequireNotInSpotNodeTerm Use this method to get Terms to prevent accidental modification of constants.
func GetRequireNotInSpotNodeTerm(cloudProvider string) []corev1.NodeSelectorRequirement {
	switch cloudProvider {
	case values.CloudProviderAWS:
		return []corev1.NodeSelectorRequirement{
			karpenterRequireNotInSpotNodeTerm,
			awsRequireNotInSpotNodeTerm,
		}
	case values.CloudProviderAlibabaCloud:
		return []corev1.NodeSelectorRequirement{
			karpenterRequireNotInSpotNodeTerm,
		}
	default:
		klog.Errorf("Unsupported cloud provider: %s, when get require note in spot node terms", cloudProvider)
		return []corev1.NodeSelectorRequirement{}
	}
}
