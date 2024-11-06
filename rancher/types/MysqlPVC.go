package types

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MySQLPersistentVolumeClaim 定义MySQL的PVC结构
type MySQLPersistentVolumeClaim struct {
	metav1.TypeMeta   `yaml:",inline"`
	metav1.ObjectMeta `yaml:"metadata"`
	Spec              PVCSpec `yaml:"spec"`
}

// PVCSpec 定义PVC的规格
type PVCSpec struct {
	AccessModes []string                       `yaml:"accessModes"`
	Resources   PersistentVolumeClaimResources `yaml:"resources"`
}

// PersistentVolumeClaimResources 定义PVC的资源请求
type PersistentVolumeClaimResources struct {
	Requests ResourceList `yaml:"requests"`
}

// ResourceList 定义资源列表
type ResourceList struct {
	Storage resource.Quantity `yaml:"storage"`
}

// NewMySQLPVC 创建新的MySQL PVC实例
func NewMySQLPVC() *MySQLPersistentVolumeClaim {
	return &MySQLPersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "mysql-data",
			Labels: map[string]string{
				"app": "mysql",
			},
		},
		Spec: PVCSpec{
			AccessModes: []string{"ReadWriteOnce"},
			Resources: PersistentVolumeClaimResources{
				Requests: ResourceList{
					Storage: resource.MustParse("20Gi"),
				},
			},
		},
	}
}
