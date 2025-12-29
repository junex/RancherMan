package rancher

import (
	"fmt"
	"log"
	"strings"

	"encoding/json"

	"gopkg.in/yaml.v2"
)

type Environment struct {
	ID        string
	Name      string
	BaseURL   string
	Project   string
	Ip        string
	username  string
	password  string
	nginxList []NginxMap
}

type NginxMap struct {
	Name     string
	BaseUrl  string
	ConfPath string
}

type jumpHostProgressListener struct {
	task *Task
	db   *DatabaseManager
}

// NewJumpHostProgressListener 创建跳板机进度监听器
func NewJumpHostProgressListener(task *Task, db *DatabaseManager) ProgressListener {
	return &jumpHostProgressListener{
		task: task,
		db:   db,
	}
}

func (l *jumpHostProgressListener) OnProgress(currentFolder string, current, total int) {
	// 直接更新 UI
	l.task.Description = fmt.Sprintf("更新跳板机 正在扫描: %d/%d  当前目录:%s", current, total, currentFolder)
}

func (l *jumpHostProgressListener) OnComplete() {
	// 直接更新 UI
	l.task.Description = fmt.Sprint("更新跳板机: 完成")
}

func (l *jumpHostProgressListener) OnBatchResult(configs []SSHUploadConfig) {
	// 将 SSHUploadConfig 转换为 UploadConfig
	var uploadConfigs []UploadConfig
	for _, config := range configs {
		uploadConfig := UploadConfig{
			Dir:    config.Dir,
			Script: config.Script,
			Jar:    config.Jar,
			Image:  config.Image,
		}
		uploadConfigs = append(uploadConfigs, uploadConfig)
	}

	// 插入数据库
	l.db.InsertUploadConfigs(uploadConfigs)
}

func LoadConfigFromDb(db *DatabaseManager) (map[string]interface{}, error) {
	configContent, err := db.GetConfigContent(1)
	if err != nil {
		return make(map[string]interface{}), err
	}
	if configContent == "" {
		return make(map[string]interface{}), nil
	}
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(configContent), &config); err != nil {
		fmt.Printf("从数据库解析配置时出错: %v\n", err)
		return make(map[string]interface{}), err
	}
	return config, nil
}

func SaveConfigToDb(db *DatabaseManager, content string) {
	db.DeleteConfig(1)
	db.InsertConfig(1, content)
}

func UpdateEnvironment(db *DatabaseManager, envName string, environment *Environment, forceUpdate bool) {
	workloadCount, _ := db.GetWorkloadCountByEnvironment(environment.ID)
	update := forceUpdate
	if workloadCount == 0 {
		update = true
	}
	if update {
		// 更新namespace
		var namespaceDBList []Namespace
		db.DeleteNamespaceByEnvironment(envName)
		allNamespaces, _ := GetNamespaceList(*environment)
		var namespaceList []NamespaceResp
		for _, ns := range allNamespaces {
			if ns.ProjectId == environment.Project {
				namespaceList = append(namespaceList, ns)
			}
		}
		for _, namespace := range namespaceList {
			namespaceDBList = append(namespaceDBList, Namespace{
				Name:        namespace.Name,
				Environment: environment.ID,
				Project:     namespace.ProjectId,
				Description: namespace.Description,
			})
		}
		db.InsertNamespaces(namespaceDBList)
		// 更新workload
		db.DeleteWorkloadByEnv(envName)
		// Get nginx reverse proxy list
		var nginxProxyList []ConfigEntry
		for _, nginxConfig := range environment.nginxList {
			nginxConf, _ := GetConfigMaps(*environment, nginxConfig.ConfPath)

			configList, _ := ParseNginxConfig(nginxConfig.BaseUrl, nginxConf)
			nginxProxyList = append(nginxProxyList, configList...)
		}
		lookupDict := CreateLookupDict(nginxProxyList)
		workloadList, _ := GetWorkloadList(*environment)

		var workloadsDBList []Workload
		for _, workload := range workloadList {
			var image, imagePullPolicy, containerEnvironment, remark string
			for _, container := range workload.Containers {
				// 只取第一个容器的信息
				if image == "" && imagePullPolicy == "" && containerEnvironment == "" {
					image = container.Image
					imagePullPolicy = container.ImagePullPolicy
					if envData, err := json.Marshal(container.Environment); err == nil {
						containerEnvironment = string(envData)
					}
				}
				// 如果容器名为 "sftp"，检查 Command 并赋值给 remark
				if container.Name == "sftp" && len(container.Command) > 0 {
					remark = "sftp账号密码: " + container.Command[0]
				}
			}

			accessPath := LookupService(lookupDict, workload.Name, workload.NamespaceID)
			workloadsDBList = append(workloadsDBList, Workload{
				Environment:          environment.ID,
				Namespace:            workload.NamespaceID,
				ProjectId:            workload.ProjectID,
				Name:                 workload.Name,
				Image:                image,
				ImagePullPolicy:      imagePullPolicy,
				ContainerEnvironment: containerEnvironment,
				AccessPath:           accessPath,
				Remark:               remark,
			})
		}
		db.InsertWorkloads(workloadsDBList)
	}
}

func UpdateService(db *DatabaseManager, environment *Environment) {
	// 删除旧的pod数据
	db.DeleteServiceByEnvironment(environment.ID)

	// 获取所有pod
	serviceList, err := GetServiceList(*environment)
	if err != nil {
		fmt.Printf("获取服务列表失败: %v\n", err)
		return
	}

	var servicesDBList []Service
	for _, service := range serviceList {
		for _, port := range service.Ports {
			workloadId := ""
			if len(service.TargetWorkloadIds) == 1 {
				parts := strings.Split(service.TargetWorkloadIds[0], ":")
				workloadId = parts[len(parts)-1] // 获取最后一个元素
			}
			servicesDBList = append(servicesDBList, Service{
				Environment:  environment.ID,
				ProjectId:    service.ProjectId,
				NamespaceId:  service.NamespaceId,
				Name:         service.Name,
				WorkloadId:   workloadId,
				Kind:         service.Kind,
				PortName:     port.Name,
				PortProtocol: port.Protocol,
				Port:         port.Port,
				TargetPort:   port.TargetPort,
				NodePort:     port.NodePort,
			})
		}
	}

	// 插入新的pod数据
	if err := db.InsertServices(servicesDBList); err != nil {
		fmt.Printf("插入服务数据失败: %v\n", err)
		return
	}
}

func UpdatePod(db *DatabaseManager, environment *Environment) {
	// 删除旧的pod数据
	db.DeletePodByEnvironment(environment.ID)

	// 获取所有pod
	podList, err := GetPodList(*environment)
	if err != nil {
		fmt.Printf("获取Pod列表失败: %v\n", err)
		return
	}

	var podsDBList []Pod
	for _, pod := range podList {
		podsDBList = append(podsDBList, Pod{
			Environment: environment.ID,
			ProjectId:   pod.ProjectId,
			NamespaceId: pod.NamespaceId,
			WorkloadId:  pod.WorkloadId,
			State:       pod.State,
		})
	}

	// 插入新的pod数据
	if err := db.InsertPods(podsDBList); err != nil {
		fmt.Printf("插入Pod数据失败: %v\n", err)
		return
	}
}

// UpdateJumpHostConfig 扫描跳板机配置
func UpdateJumpHostConfig(db *DatabaseManager, task *Task) bool {
	log.Printf("[UpdateJumpHostConfig] 更新跳板机")
	db.DeleteAllUploadConfigs()
	ListUploadConfig(task.JumpHostConfig, 50, NewJumpHostProgressListener(task, db))
	return true
}

func GetEnvironmentFromConfig(config map[string]interface{}, envName string) (*Environment, error) {
	// 从配置中获取environments部分
	environments, ok := config["environment"].(map[interface{}]interface{})
	if !ok {
		fmt.Println("配置中找不到environment部分")
		return nil, fmt.Errorf("配置中找不到environment部分")
	}

	// 查找指定的环境
	for name, envData := range environments {
		if name.(string) == envName {
			env := envData.(map[interface{}]interface{})
			key := env["key"].(map[interface{}]interface{})

			// 解析nginx配置
			var nginxConfigs []NginxMap
			if nginxData, exists := env["nginx"].(map[interface{}]interface{}); exists {
				for Name, nginxConfig := range nginxData {
					nginx := nginxConfig.(map[interface{}]interface{})
					nginxConfigs = append(nginxConfigs, NginxMap{
						Name:     Name.(string),
						BaseUrl:  nginx["base_url"].(string),
						ConfPath: nginx["nginx_conf"].(string),
					})
				}
			}

			return &Environment{
				ID:        name.(string),
				Name:      env["name"].(string),
				BaseURL:   env["base_url"].(string),
				Project:   env["project"].(string),
				Ip:        env["ip"].(string),
				username:  key["name"].(string),
				password:  key["token"].(string),
				nginxList: nginxConfigs,
			}, nil
		}
	}

	fmt.Printf("找不到环境: %s\n", envName)
	return nil, fmt.Errorf("找不到环境: %s", envName)
}
