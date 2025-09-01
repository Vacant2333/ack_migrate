package utils

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/cloudpilot-ai/cloudpilot-agent/pkg/definitions"
)

func PatchPodSpotNodeAffinity(pod *corev1.Pod, cloudProvider string) {
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}
	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}
	if pod.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution == nil {
		pod.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = []corev1.PreferredSchedulingTerm{}
	}
	pod.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution =
		append(pod.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
			definitions.GetPreferSpotNodeTerms(cloudProvider)...)
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	pod.Annotations[corev1.PodDeletionCost] = "-1"
}

func PatchPodNonSpotNodeAffinity(pod *corev1.Pod, cloudProvider string) {
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}
	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}
	if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{},
		}
	}

	if len(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) == 0 {
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = []corev1.NodeSelectorTerm{{
			MatchExpressions: definitions.GetRequireNotInSpotNodeTerm(cloudProvider),
		}}
	} else {
		for index, term := range pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
			term.MatchExpressions = append(term.MatchExpressions, definitions.GetRequireNotInSpotNodeTerm(cloudProvider)...)
			pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[index] = term
		}
	}
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	pod.Annotations[corev1.PodDeletionCost] = "1"
}

func PatchDiversityPodAntiAffinity(pod *corev1.Pod, instanceType string) {
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}
	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}
	if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{},
		}
	}

	antiAffinityForOptimizer := []corev1.NodeSelectorRequirement{
		{
			Key:      corev1.LabelInstanceType,
			Operator: corev1.NodeSelectorOpNotIn,
			Values:   []string{instanceType},
		},
	}
	if len(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) == 0 {
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = []corev1.NodeSelectorTerm{{
			MatchExpressions: antiAffinityForOptimizer,
		}}
	} else {
		for index, term := range pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
			// Overwrite or append
			isExists := false
			for _, matchExpression := range term.MatchExpressions {
				if matchExpression.Key == corev1.LabelInstanceType && matchExpression.Operator == corev1.NodeSelectorOpNotIn {
					isExists = true
					break
				}
			}
			if !isExists {
				term.MatchExpressions = append(term.MatchExpressions, antiAffinityForOptimizer...)
			}

			pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[index] = term
		}
	}
}
