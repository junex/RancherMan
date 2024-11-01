package rancher

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type Environment struct {
	ID       string
	Name     string
	BaseURL  string
	Project  string
	username string
	password string
}

func loadConfig(db *DatabaseManager) map[string]interface{} {
	configContent, _ := db.GetConfigContent(1)
	if configContent == "" {
		fmt.Println("读取配置文件: config.yml")
		content, err := os.ReadFile("config.yml")
		if err != nil {
			panic(err)
		}
		configContent = string(content)
		db.DeleteConfig(1)
		db.InsertConfig(1, configContent)
	}
	var config map[string]interface{}
	err := yaml.Unmarshal([]byte(configContent), &config)
	if err != nil {
		panic(err)
	}
	return config
}

func UpdateEnvironment(db *DatabaseManager, config map[string]interface{}, forceUpdate bool) {
	for envName, envData := range config["environment"].(map[interface{}]interface{}) {
		env := envData.(map[interface{}]interface{})
		workloadCount, _ := db.GetWorkloadCountByEnvironment(envName.(string))
		update := forceUpdate
		if workloadCount == 0 {
			update = true
		}
		if update {
			environment, _ := GetEnvironmentFromConfig(config, envName.(string))
			db.DeleteWorkloadByEnv(envName.(string))

			// Get nginx reverse proxy list
			var nginxProxyList []ConfigEntry
			for _, nginxConfig := range env["nginx"].(map[interface{}]interface{}) {
				nginx := nginxConfig.(map[interface{}]interface{})
				serviceBaseURL := nginx["base_url"].(string)
				confPath := nginx["nginx_conf"].(string)
				nginxConf, _ := GetConfigMaps(*environment, confPath)

				configList, _ := ParseNginxConfig(serviceBaseURL, nginxConf)
				nginxProxyList = append(nginxProxyList, configList...)
			}
			lookupDict := CreateLookupDict(nginxProxyList)
			workloadList, _ := GetWorkloadsList(*environment)

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
					Environment: envName.(string),
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
}

func GetEnvironmentFromConfig(config map[string]interface{}, envName string) (*Environment, error) {
	// 从配置中获取environments部分
	environments, ok := config["environment"].(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("配置中找不到environment部分")
	}

	// 查找指定的环境
	for name, envData := range environments {
		if name.(string) == envName {
			env := envData.(map[interface{}]interface{})
			key := env["key"].(map[interface{}]interface{})

			return &Environment{
				ID:       name.(string),
				Name:     env["name"].(string),
				BaseURL:  env["base_url"].(string),
				Project:  env["project"].(string),
				username: key["name"].(string),
				password: key["token"].(string),
			}, nil
		}
	}

	return nil, fmt.Errorf("找不到环境: %s", envName)
}
