package rancher

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"gorm.io/gorm/utils"

	"gopkg.in/yaml.v2"
)

type RancherUtils struct {
	config      map[string]interface{}
	db          *DatabaseManager
	environment *Environment
	namespace   string
}

type Environment struct {
	ID       string
	Name     string
	BaseURL  string
	Project  string
	username string
	password string
}

type WorkloadResp struct {
	Name            string
	NamespaceID     string
	Containers      []Container
	PublicEndpoints []Endpoint
}

type Container struct {
	Image string
}

type Endpoint struct {
	Port int
}

func NewRancherUtils(manager *DatabaseManager) *RancherUtils {
	ru := &RancherUtils{
		db: manager,
	}
	ru.loadConfig()
	return ru
}

func (ru *RancherUtils) loadConfig() {
	configContent, _ := ru.db.GetConfigContent(1)
	if configContent == "" {
		fmt.Println("读取配置文件: config.yml")
		content, err := os.ReadFile("config.yml")
		if err != nil {
			panic(err)
		}
		configContent = string(content)
		ru.db.DeleteConfig(1)
		ru.db.InsertConfig(1, configContent)
	}

	err := yaml.Unmarshal([]byte(configContent), &ru.config)
	if err != nil {
		panic(err)
	}
}

func (ru *RancherUtils) UpdateEnvironment(forceUpdate bool) {

	for envName, envData := range ru.config["environment"].(map[interface{}]interface{}) {
		env := envData.(map[interface{}]interface{})
		workloadCount, _ := ru.db.GetWorkloadCountByEnvironment(envName.(string))
		update := forceUpdate
		if workloadCount == 0 {
			update = true
		}
		if update {
			fmt.Printf("更新")
			ru.UseNamespace("", envName.(string))
			ru.db.DeleteWorkloadByEnv(envName.(string))

			// Get nginx reverse proxy list
			var nginxProxyList []ConfigEntry
			for _, nginxConfig := range env["nginx"].(map[interface{}]interface{}) {
				nginx := nginxConfig.(map[interface{}]interface{})
				serviceBaseURL := nginx["base_url"].(string)
				confPath := nginx["nginx_conf"].(string)

				resp, err := ru.makeRequest("GET", fmt.Sprintf("configMaps/%s", confPath), nil)
				if err != nil {
					log.Printf("Error fetching nginx config: %v", err)
					continue
				}

				var configMap struct {
					Data struct {
						DefaultConf string `json:"default.conf"`
					} `json:"data"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&configMap); err != nil {
					log.Printf("Error decoding nginx config: %v", err)
					continue
				}
				resp.Body.Close()

				configList, err := ParseNginxConfig(serviceBaseURL, configMap.Data.DefaultConf)
				nginxProxyList = append(nginxProxyList, configList...)
			}

			lookupDict := CreateLookupDict(nginxProxyList)

			// Get workloads list
			resp, err := ru.makeRequest("GET", "workloads?limit=-1", nil)
			if err != nil {
				log.Printf("Error fetching workloads: %v", err)
				continue
			}

			var workloadsResponse struct {
				Data []WorkloadResp `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&workloadsResponse); err != nil {
				log.Printf("Error decoding workloads: %v", err)
				continue
			}
			resp.Body.Close()

			var workloadsDBList []Workload
			for _, workload := range workloadsResponse.Data {
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

			ru.db.InsertWorkloads(workloadsDBList)
			fmt.Println("完成")
		}
	}
}

func (ru *RancherUtils) makeRequest(method, url string, payload []byte) (*http.Response, error) {
	baseURL := ru.environment.BaseURL
	project := ru.environment.Project
	fullURL := fmt.Sprintf("%s/project/%s/%s", baseURL, project, url)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	var body io.Reader
	if payload != nil {
		body = strings.NewReader(string(payload))
	}
	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(ru.environment.username, ru.environment.password)
	return client.Do(req)
}

func (ru *RancherUtils) UseNamespace(namespace, environmentName string) {
	ru.namespace = namespace
	if environmentName == "" {
		environmentNameList, _ := ru.db.GetEnvironmentsByNamespace(namespace)
		if len(environmentNameList) == 1 {
			environmentName = environmentNameList[0]
		} else if len(environmentNameList) > 1 {
			fmt.Printf("找到多个环境:%v\n", environmentNameList)
		}
	}

	for envName, envData := range ru.config["environment"].(map[interface{}]interface{}) {
		if envName.(string) == environmentName {
			env := envData.(map[interface{}]interface{})
			key := env["key"].(map[interface{}]interface{})
			ru.environment = &Environment{
				ID:       envName.(string),
				Name:     env["name"].(string),
				BaseURL:  env["base_url"].(string),
				Project:  env["project"].(string),
				username: key["name"].(string),
				password: key["token"].(string),
			}
			fmt.Printf("%s\n", env["name"].(string))
			return
		}
	}

	fmt.Printf("找不到环境 %s\n", environmentName)
}

func (ru *RancherUtils) Scale(workload string, replicas int) {

	service := workload
	if colonIndex := strings.LastIndex(workload, ":"); colonIndex > 0 {
		service = workload[colonIndex+1:]
	}

	fmt.Printf("%s\tscale:%d\t", service, replicas)

	payload := map[string]int{"scale": replicas}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := ru.makeRequest("PUT", fmt.Sprintf("workloads/deployment:%s:%s", ru.namespace, service), jsonPayload)

	if err != nil {
		fmt.Println("失败")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("成功")
	} else {
		fmt.Println("失败")
	}
}

func (ru *RancherUtils) Redeploy(workload string) {

	service := workload
	if colonIndex := strings.LastIndex(workload, ":"); colonIndex > 0 {
		service = workload[colonIndex+1:]
	}

	fmt.Printf("%s\tredeploy\t", service)

	resp, err := ru.makeRequest("POST", fmt.Sprintf("workloads/deployment:%s:%s?action=redeploy", ru.namespace, service), nil)
	if err != nil {
		fmt.Println("失败")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("成功")
	} else {
		fmt.Println("失败")
	}
}

func (ru *RancherUtils) ScaleNamespace(replicas int) {

	fmt.Printf("开始操作命名空间:%s\n", ru.namespace)
	ru.UpdateEnvironment(false)
	workloads, _ := ru.db.GetWorkloadNamesByEnvNamespace(ru.environment.ID, ru.namespace)
	notOpList := []string{} // Populate this list as needed

	for _, workload := range workloads {
		service := workload
		if colonIndex := strings.LastIndex(workload, ":"); colonIndex > 0 {
			service = workload[colonIndex+1:]
		}

		if !utils.Contains(notOpList, service) {
			ru.Scale(service, replicas)
		} else {
			fmt.Printf("%s\t跳过\n", service)
		}
	}

	fmt.Printf("结束操作命名空间:%s\n", ru.namespace)
}

func (ru *RancherUtils) List() {
	workloads, err := ru.db.GetWorkloadDetailsByEnvNamespace(ru.environment.ID, ru.namespace)
	if err != nil {
		fmt.Printf("Error fetching workloads: %v\n", err)
		return
	}

	for _, workload := range workloads {
		fmt.Printf("%s\t%s\n", workload.Name, workload.Image)
		if workload.NodePort != "" {
			fmt.Printf("%s\t", workload.NodePort)
		}
		if workload.AccessPath != "" {
			fmt.Print(workload.AccessPath)
		}
		fmt.Println()
	}
}
