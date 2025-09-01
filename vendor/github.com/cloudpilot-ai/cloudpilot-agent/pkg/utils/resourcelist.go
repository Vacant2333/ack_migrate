package utils

import (
	v1 "k8s.io/api/core/v1"
)

func SumResourceRequests(containers []v1.Container) v1.ResourceList {
	totalRequests := v1.ResourceList{}
	for _, container := range containers {
		for resourceName, quantity := range container.Resources.Requests {
			if total, ok := totalRequests[resourceName]; ok {
				total.Add(quantity)
				totalRequests[resourceName] = total
				continue
			}
			totalRequests[resourceName] = quantity.DeepCopy()
		}
	}
	return totalRequests
}

// ResourceListEquals compares two v1.ResourceList instances and returns true if they are equal.
func ResourceListEquals(a, b v1.ResourceList) bool {
	for key, aValue := range a {
		if bValue, exists := b[key]; !exists || !aValue.Equal(bValue) {
			return false
		}
	}
	for key := range b {
		if _, exists := a[key]; !exists {
			return false
		}
	}
	return true
}

// AddResourceListsInPlace adds the contents of list2 to list1, modifying list1.
func AddResourceListsInPlace(list1, list2 v1.ResourceList) {
	for key, value2 := range list2 {
		if value1, exists := list1[key]; exists {
			value1.Add(value2)
			list1[key] = value1
			continue
		}
		list1[key] = value2.DeepCopy()
	}
}

func SubResourceListsInPlace(list1, list2 v1.ResourceList) {
	for key, value2 := range list2 {
		if value1, exists := list1[key]; exists {
			value1.Sub(value2)
			list1[key] = value1
		}
	}
}
