package workload

// Deployment 表示 Kubernetes Deployment 资源
type Deployment struct {
	ApiVersion string   `yaml:"apiVersion" json:"apiVersion"`
	Kind       string   `yaml:"kind" json:"kind"`
	Metadata   Metadata `yaml:"metadata" json:"metadata"`
	Spec       Spec     `yaml:"spec" json:"spec"`
}

// Metadata 表示资源元数据
type Metadata struct {
	Name      string `yaml:"name" json:"name"`
	Namespace string `yaml:"namespace" json:"namespace"`
}

// Spec 表示 Deployment 的规格
type Spec struct {
	Selector Selector `yaml:"selector" json:"selector"`
	Strategy Strategy `yaml:"strategy,omitempty" json:"strategy,omitempty"`
	Template Template `yaml:"template" json:"template"`
}

// Selector 表示标签选择器
type Selector struct {
	MatchLabels map[string]string `yaml:"matchLabels" json:"matchLabels"`
}

// Strategy 表示部署策略
type Strategy struct {
	Type string `yaml:"type,omitempty" json:"type,omitempty"`
}

// Template 表示 Pod 模板
type Template struct {
	Metadata PodMetadata `yaml:"metadata" json:"metadata"`
	Spec     PodSpec     `yaml:"spec" json:"spec"`
}

// PodMetadata 表示 Pod 元数据
type PodMetadata struct {
	Labels map[string]string `yaml:"labels" json:"labels"`
}

// PodSpec 表示 Pod 规格
type PodSpec struct {
	HostAliases      []HostAlias  `yaml:"hostAliases,omitempty" json:"hostAliases,omitempty"`
	Affinity         Affinity     `yaml:"affinity" json:"affinity"`
	Containers       []Container  `yaml:"containers" json:"containers"`
	ImagePullSecrets []PullSecret `yaml:"imagePullSecrets" json:"imagePullSecrets"`
	SchedulerName    string       `yaml:"schedulerName" json:"schedulerName"`
	Volumes          []Volume     `yaml:"volumes" json:"volumes"`
}

type HostAlias struct {
	IP        string   `yaml:"ip" json:"ip"`
	Hostnames []string `yaml:"hostnames" json:"hostnames"`
}

// Affinity 表示节点亲和性
type Affinity struct {
	NodeAffinity NodeAffinity `yaml:"nodeAffinity" json:"nodeAffinity"`
}

// NodeAffinity 表示节点亲和性规则
type NodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution NodeSelector `yaml:"requiredDuringSchedulingIgnoredDuringExecution" json:"requiredDuringSchedulingIgnoredDuringExecution"`
}

// NodeSelector 表示节点选择器
type NodeSelector struct {
	NodeSelectorTerms []NodeSelectorTerm `yaml:"nodeSelectorTerms" json:"nodeSelectorTerms"`
}

// NodeSelectorTerm 表示节点选择条件
type NodeSelectorTerm struct {
	MatchExpressions []MatchExpression `yaml:"matchExpressions" json:"matchExpressions"`
}

// MatchExpression 表示匹配表达式
type MatchExpression struct {
	Key      string   `yaml:"key" json:"key"`
	Operator string   `yaml:"operator" json:"operator"`
	Values   []string `yaml:"values" json:"values"`
}

// Container 表示容器配置
type Container struct {
	Name            string           `yaml:"name" json:"name"`
	Image           string           `yaml:"image" json:"image"`
	Args            []string         `yaml:"args,omitempty" json:"args,omitempty"`
	Ports           []Port           `yaml:"ports,omitempty" json:"ports,omitempty"`
	Env             []EnvVar         `yaml:"env,omitempty" json:"env,omitempty"`
	ImagePullPolicy string           `yaml:"imagePullPolicy,omitempty" json:"imagePullPolicy,omitempty"`
	VolumeMounts    []VolumeMount    `yaml:"volumeMounts,omitempty" json:"volumeMounts,omitempty"`
	SecurityContext *SecurityContext `yaml:"securityContext,omitempty" json:"securityContext,omitempty"`
}

type VolumeMount struct {
	Name      string `yaml:"name" json:"name"`
	MountPath string `yaml:"mountPath" json:"mountPath"`
	SubPath   string `yaml:"subPath,omitempty" json:"subPath,omitempty"`
}

type SecurityContext struct {
	AllowPrivilegeEscalation *bool `yaml:"allowPrivilegeEscalation,omitempty" json:"allowPrivilegeEscalation,omitempty"`
	Privileged               *bool `yaml:"privileged,omitempty" json:"privileged,omitempty"`
	ReadOnlyRootFilesystem   *bool `yaml:"readOnlyRootFilesystem,omitempty" json:"readOnlyRootFilesystem,omitempty"`
	RunAsNonRoot             *bool `yaml:"runAsNonRoot,omitempty" json:"runAsNonRoot,omitempty"`
}

// Port 表示容器端口
type Port struct {
	ContainerPort int32  `yaml:"containerPort" json:"containerPort"`
	Protocol      string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
	Name          string `yaml:"name,omitempty" json:"name,omitempty"`
}

// EnvVar 表示环境变量
type EnvVar struct {
	Name  string `yaml:"name" json:"name"`
	Value string `yaml:"value" json:"value"`
}

// PullSecret 表示镜像拉取密钥
type PullSecret struct {
	Name string `yaml:"name" json:"name"`
}

// Volume 表示数据卷配置
type Volume struct {
	Name                  string                 `yaml:"name" json:"name"`
	ConfigMap             *ConfigMap             `yaml:"configMap,omitempty" json:"configMap,omitempty"`
	PersistentVolumeClaim *PersistentVolumeClaim `yaml:"persistentVolumeClaim,omitempty" json:"persistentVolumeClaim,omitempty"`
}

// ConfigMap 表示 ConfigMap 卷配置
type ConfigMap struct {
	DefaultMode *int32 `yaml:"defaultMode,omitempty" json:"defaultMode,omitempty"`
	Name        string `yaml:"name" json:"name"`
	Optional    *bool  `yaml:"optional,omitempty" json:"optional,omitempty"`
}

type PersistentVolumeClaim struct {
	ClaimName string `yaml:"claimName" json:"claimName"`
}
