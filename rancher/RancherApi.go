package rancher

import (
	"RancherMan/rancher/types/configMaps"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type WorkloadResp struct {
	Name        string
	NamespaceID string
	ProjectID   string
	Containers  []Container
}

type Container struct {
	Image           string
	ImagePullPolicy string
	Environment     map[string]string
}

type NamespaceResp struct {
	Name        string
	ProjectId   string
	Description string
}

type PodResp struct {
	ProjectId   string
	NamespaceId string
	WorkloadId  string
	State       string
}

type ServiceResp struct {
	ProjectId         string
	NamespaceId       string
	Name              string
	TargetWorkloadIds []string
	Kind              string
	Ports             []PortResp
}

type PortResp struct {
	Name       string
	Protocol   string
	Port       int
	TargetPort int
	NodePort   int
}

func makeProjectRequest(environment Environment, method, url string, payload []byte) (*http.Response, error) {
	project := environment.Project
	fullURL := fmt.Sprintf("project/%s/%s", project, url)
	return makeRequest(environment, method, fullURL, payload, "")
}

func makeRequest(environment Environment, method, url string, payload []byte, accept string) (*http.Response, error) {
	baseURL := environment.BaseURL
	fullURL := fmt.Sprintf("%s/%s", baseURL, url)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	var body io.Reader
	if payload != nil {
		body = strings.NewReader(string(payload))
	}
	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(environment.username, environment.password)
	return client.Do(req)
}

func Scale(environment Environment, namespace string, workload string, replicas int) bool {

	service := workload
	if colonIndex := strings.LastIndex(workload, ":"); colonIndex > 0 {
		service = workload[colonIndex+1:]
	}

	payload := map[string]int{"scale": replicas}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := makeProjectRequest(environment, "PUT", fmt.Sprintf("workloads/deployment:%s:%s", namespace, service), jsonPayload)

	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func Redeploy(environment Environment, namespace string, workload string) bool {

	service := workload
	if colonIndex := strings.LastIndex(workload, ":"); colonIndex > 0 {
		service = workload[colonIndex+1:]
	}

	resp, err := makeProjectRequest(environment, "POST", fmt.Sprintf("workloads/deployment:%s:%s?action=redeploy", namespace, service), nil)
	if err != nil {
		fmt.Println("失败")
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func GetConfigMaps(environment Environment, confPath string) (string, error) {
	resp, err := makeProjectRequest(environment, "GET", fmt.Sprintf("configMaps/%s", confPath), nil)
	if err != nil {
		log.Printf("Error fetching nginx config: %v", err)
		return "", err
	}

	var configMap struct {
		Data struct {
			DefaultConf string `json:"default.conf"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&configMap); err != nil {
		log.Printf("Error decoding nginx config: %v", err)
		return "", err
	}
	resp.Body.Close()
	return configMap.Data.DefaultConf, nil
}

func GetWorkloadList(environment Environment) ([]WorkloadResp, error) {

	resp, err := makeProjectRequest(environment, "GET", "workloads?limit=-1", nil)
	if err != nil {
		log.Printf("Error fetching workloads: %v", err)
		return nil, err
	}

	var workloadsResponse struct {
		Data []WorkloadResp `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&workloadsResponse); err != nil {
		log.Printf("Error decoding workloads: %v", err)
		return nil, err
	}
	resp.Body.Close()
	return workloadsResponse.Data, nil
}

func GetNamespaceList(environment Environment) ([]NamespaceResp, error) {

	resp, err := makeRequest(environment, "GET", "cluster/local/namespaces?limit=-1", nil, "")
	if err != nil {
		log.Printf("Error fetching namespace: %v", err)
		return nil, err
	}

	var NamespaceResponse struct {
		Data []NamespaceResp `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&NamespaceResponse); err != nil {
		log.Printf("Error decoding namespace: %v", err)
		return nil, err
	}
	resp.Body.Close()
	return NamespaceResponse.Data, nil
}

func GetPodList(environment Environment) ([]PodResp, error) {

	resp, err := makeProjectRequest(environment, "GET", "pods?limit=-1", nil)
	if err != nil {
		log.Printf("Error fetching pods: %v", err)
		return nil, err
	}

	var podsResponse struct {
		Data []PodResp `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&podsResponse); err != nil {
		log.Printf("Error decoding pods: %v", err)
		return nil, err
	}
	resp.Body.Close()
	return podsResponse.Data, nil
}

func GetDeploymentYaml(environment Environment, namespace string, workload string) (string, error) {
	project := environment.Project
	url := fmt.Sprintf("project/%s/workloads/deployment:%s:%s/yaml?export=true", project, namespace, workload)
	response, err := makeRequest(environment, "GET", url, nil, "application/yaml")
	if err != nil {
		log.Printf("Error fetching workloads: %v", err)
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return "", err
	}

	return string(body), nil
}

func GetConfigMapList(environment Environment, namespace string) ([]configMaps.ConfigMap, error) {
	resp, err := makeProjectRequest(environment, "GET", fmt.Sprintf("configMap?namespaceId=%s&limit=-1", namespace), nil)
	if err != nil {
		log.Printf("Error fetching configMap: %v", err)
		return nil, err
	}

	var configMapsResponse struct {
		Data []configMaps.ConfigMap `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&configMapsResponse); err != nil {
		log.Printf("Error decoding configMap: %v", err)
		return nil, err
	}
	resp.Body.Close()
	return configMapsResponse.Data, nil
}

func ImportYaml(environment Environment, defaultNamespace string, yaml []byte) error {
	payload := map[string]string{
		"yaml":             string(yaml),
		"defaultNamespace": defaultNamespace,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling yaml payload: %v", err)
		return err
	}
	response, err := makeRequest(environment, "POST", "clusters/local?action=importYaml", jsonPayload, "")
	if err != nil {
		log.Printf("Error importing yaml: %v", err)
		return err
	}
	defer response.Body.Close()
	return nil
}

func GetServiceList(environment Environment) ([]ServiceResp, error) {
	resp, err := makeProjectRequest(environment, "GET", "services?limit=-1", nil)
	if err != nil {
		log.Printf("Error fetching services: %v", err)
		return nil, err
	}

	var servicesResponse struct {
		Data []ServiceResp `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&servicesResponse); err != nil {
		log.Printf("Error decoding services: %v", err)
		return nil, err
	}
	resp.Body.Close()
	return servicesResponse.Data, nil
}
