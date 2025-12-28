package operations

import (
	"strings"

	"RancherMan/rancher"
)

// FilterNamespaces 过滤命名空间列表
func FilterNamespaces(items []rancher.Namespace, filter string) []rancher.Namespace {
	if filter == "" {
		return items
	}
	var filtered []rancher.Namespace
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), strings.ToLower(filter)) || strings.Contains(strings.ToLower(item.Description), strings.ToLower(filter)) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// FilterWorkloads 过滤工作负载列表
func FilterWorkloads(items []rancher.Workload, filter string) []rancher.Workload {
	if filter == "" {
		return items
	}
	var filtered []rancher.Workload
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), strings.ToLower(filter)) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
