package rancher

import (
	"fmt"
	"strconv"
	"strings"

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
	workloadCount, _ := db.GetWorkloadCountByEnvironment(environment.Name)
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
				Environment: envName,
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
			var image, nodePort string
			if len(workload.Containers) == 1 {
				image = workload.Containers[0].Image
			}
			if len(workload.PublicEndpoints) > 0 {
				ports := make([]string, len(workload.PublicEndpoints))
				for i, endpoint := range workload.PublicEndpoints {
					ports[i] = strconv.Itoa(endpoint.Port)
				}
				nodePort = strings.Join(ports, ",")
			}
			accessPath := LookupService(lookupDict, workload.Name, workload.NamespaceID)
			workloadsDBList = append(workloadsDBList, Workload{
				Environment: envName,
				Namespace:   workload.NamespaceID,
				Name:        workload.Name,
				Image:       image,
				NodePort:    nodePort,
				AccessPath:  accessPath,
			})
		}
		db.InsertWorkloads(workloadsDBList)
	}
}

func UpdatePod(db *DatabaseManager, envName string, environment *Environment) {
	// 删除旧的pod数据
	db.DeletePodByEnvironment(envName)

	// 获取所有pod
	podList, err := GetPodList(*environment)
	if err != nil {
		fmt.Printf("获取Pod列表失败: %v\n", err)
		return
	}

	var podsDBList []Pod
	for _, pod := range podList {
		podsDBList = append(podsDBList, Pod{
			Environment: envName,
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
