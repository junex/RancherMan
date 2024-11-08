package configMaps

import (
	"RancherMan/rancher/types/workload"
)

// ConfigMap 表示Kubernetes的ConfigMap资源
type ConfigMap struct {
	ApiVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   workload.Metadata `yaml:"metadata"`
	Data       map[string]string `yaml:"data" json:"data"` // 修改为map类型，支持任意键值对
	Name       string            `yaml:"-" json:"name"`
}
