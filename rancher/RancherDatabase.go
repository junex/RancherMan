package rancher

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Workload 工作负载模型
type Workload struct {
	ID                   uint   `gorm:"primaryKey"`
	Environment          string `gorm:"size:20"`
	ProjectId            string `gorm:"size:20"`
	Namespace            string `gorm:"size:50"`
	Name                 string `gorm:"size:30"`
	Image                string `gorm:"size:100"`
	ImagePullPolicy      string `gorm:"size:20"`
	ContainerEnvironment string `gorm:"size:255"`
	AccessPath           string `gorm:"size:500"`
}

func (Workload) TableName() string {
	return "workload"
}

// Config 配置模型
type Config struct {
	ID      uint   `gorm:"primaryKey"`
	Content string `gorm:"type:text"`
}

func (Config) TableName() string {
	return "config"
}

// Namespace 命名空间模型
type Namespace struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"size:30"`
	Project     string `gorm:"size:30"`
	Environment string `gorm:"size:20"`
	Description string `gorm:"size:30"`
}

func (Namespace) TableName() string {
	return "namespace"
}

// Pod 模型
type Pod struct {
	ID          uint   `gorm:"primaryKey"`
	Environment string `gorm:"size:20"`
	ProjectId   string `gorm:"size:20"`
	NamespaceId string `gorm:"size:20"`
	WorkloadId  string `gorm:"size:80"`
	State       string `gorm:"size:10"`
}

func (Pod) TableName() string {
	return "pod"
}

// UploadConfig 上传配置模型
type UploadConfig struct {
	ID     uint   `gorm:"primaryKey"`
	Dir    string `gorm:"size:100"`
	Script string `gorm:"size:30"`
	Jar    string `gorm:"size:50"`
	Image  string `gorm:"size:50"`
}

func (UploadConfig) TableName() string {
	return "upload_config"
}

// Service 服务模型
type Service struct {
	ID           uint   `gorm:"primaryKey"`
	Environment  string `gorm:"size:20"`
	ProjectId    string `gorm:"size:20"`
	NamespaceId  string `gorm:"size:20"`
	Name         string `gorm:"size:20"`
	WorkloadId   string `gorm:"size:40"`
	Kind         string `gorm:"size:10"`
	PortName     string `gorm:"size:50"`
	PortProtocol string `gorm:"size:10"`
	Port         int
	TargetPort   int
	NodePort     int
}

func (Service) TableName() string {
	return "service"
}

// DatabaseManager 数据库管理器结构体
type DatabaseManager struct {
	db     *gorm.DB
	dbFile string
}

// NewDatabaseManager 创建新的数据库管理器实例
func NewDatabaseManager(dbFile string) (*DatabaseManager, error) {
	if dbFile == "" {
		// 获取 APPDATA 目录
		appData := os.Getenv("APPDATA")
		if appData == "" {
			// 如果 APPDATA 不存在（非 Windows 系统），使用用户主目录
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("无法获取用户主目录: %v", err)
			}
			appData = filepath.Join(homeDir, ".config")
		}

		// 创建应用数据目录（如果不存在）
		appDir := filepath.Join(appData, "Rancher助手")
		if err := os.MkdirAll(appDir, 0755); err != nil {
			return nil, fmt.Errorf("创建应用数据目录失败: %v", err)
		}

		dbFile = filepath.Join(appDir, "app.db")
	}

	fmt.Printf("使用数据库文件: %s\n", dbFile)

	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %v", err)
	}

	dm := &DatabaseManager{
		db:     db,
		dbFile: dbFile,
	}

	// 自动迁移数据库结构
	if err := dm.initDatabase(); err != nil {
		return nil, err
	}

	return dm, nil
}

// Close 关闭数据库连接
func (dm *DatabaseManager) Close() error {
	sqlDB, err := dm.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// initDatabase 初始化数据库，创建必要的表
func (dm *DatabaseManager) initDatabase() error {
	return dm.db.AutoMigrate(&Workload{}, &Config{}, &Namespace{}, &Pod{}, &UploadConfig{}, &Service{})
}

// GetWorkloadDetailsByEnvNamespace 根据环境和命名空间获取工作负载详细信息
func (dm *DatabaseManager) GetWorkloadDetailsByEnvNamespace(environment, namespace string) ([]Workload, error) {
	var workloads []Workload
	result := dm.db.Where("environment = ? AND namespace = ?", environment, namespace).Find(&workloads)
	return workloads, result.Error
}

// GetWorkloadNamesByEnvNamespace 根据环境和命名空间获取工作负载名称列表
func (dm *DatabaseManager) GetWorkloadNamesByEnvNamespace(environment, namespace string) ([]string, error) {
	var names []string
	result := dm.db.Model(&Workload{}).
		Where("environment = ? AND namespace = ?", environment, namespace).
		Pluck("name", &names)
	return names, result.Error
}

// GetWorkloadCountByEnvironment 获取指定环境下的工作负载数量
func (dm *DatabaseManager) GetWorkloadCountByEnvironment(environment string) (int64, error) {
	var count int64
	result := dm.db.Model(&Workload{}).Where("environment = ?", environment).Count(&count)
	return count, result.Error
}

// DeleteWorkloadByEnvNamespace 根据环境和命名空间删除工作负载
func (dm *DatabaseManager) DeleteWorkloadByEnvNamespace(environment, namespace string) (int64, error) {
	result := dm.db.Where("environment = ? AND namespace = ?", environment, namespace).Delete(&Workload{})
	return result.RowsAffected, result.Error
}

// DeleteWorkloadByEnv 根据环境删除工作负载
func (dm *DatabaseManager) DeleteWorkloadByEnv(environment string) (int64, error) {
	result := dm.db.Where("environment = ?", environment).Delete(&Workload{})
	return result.RowsAffected, result.Error
}

// InsertWorkloads 批量插入工作负载
func (dm *DatabaseManager) InsertWorkloads(workloads []Workload) error {
	return dm.db.Transaction(func(tx *gorm.DB) error {
		for _, workload := range workloads {
			if err := tx.Create(&workload).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetEnvironmentsByNamespace 根据命名空间获取环境列表
func (dm *DatabaseManager) GetEnvironmentsByNamespace(namespace string) ([]string, error) {
	var environments []string
	result := dm.db.Model(&Workload{}).
		Where("namespace = ?", namespace).
		Distinct().
		Pluck("environment", &environments)
	return environments, result.Error
}

// GetConfigContent 根据ID获取配置内容
func (dm *DatabaseManager) GetConfigContent(id uint) (string, error) {
	var config Config
	result := dm.db.First(&config, id)
	if result.Error == gorm.ErrRecordNotFound {
		return "", nil
	}
	return config.Content, result.Error
}

// InsertConfig 插入新的配置
func (dm *DatabaseManager) InsertConfig(id uint, content string) error {
	config := Config{
		ID:      id,
		Content: content,
	}
	return dm.db.Create(&config).Error
}

// DeleteConfig 删除配置
func (dm *DatabaseManager) DeleteConfig(id uint) (bool, error) {
	result := dm.db.Delete(&Config{}, id)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// BatchCreateWorkloads 批量创建工作负载的辅助方法
func (dm *DatabaseManager) BatchCreateWorkloads(workloads []Workload) error {
	return dm.db.CreateInBatches(workloads, 100).Error
}

// UpdateWorkload 更新工作负载信息
func (dm *DatabaseManager) UpdateWorkload(workload *Workload) error {
	return dm.db.Save(workload).Error
}

// GetWorkloadByID 根据ID获取工作负载
func (dm *DatabaseManager) GetWorkloadByID(id uint) (*Workload, error) {
	var workload Workload
	result := dm.db.First(&workload, id)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &workload, result.Error
}

// GetWorkloadsByNamespace 根据命��空间获取工作负载列表
func (dm *DatabaseManager) GetWorkloadsByNamespace(namespace string) ([]Workload, error) {
	var workloads []Workload
	result := dm.db.Where("namespace = ?", namespace).Find(&workloads)
	return workloads, result.Error
}

// DeleteNamespaceByEnvironment 根据环境删除命名空间数据
func (dm *DatabaseManager) DeleteNamespaceByEnvironment(environment string) (int64, error) {
	result := dm.db.Where("environment = ?", environment).Delete(&Namespace{})
	return result.RowsAffected, result.Error
}

// InsertNamespaces 批量插入命名空间数据
func (dm *DatabaseManager) InsertNamespaces(namespaces []Namespace) error {
	return dm.db.Transaction(func(tx *gorm.DB) error {
		for _, namespace := range namespaces {
			if err := tx.Create(&namespace).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetAllNamespacesDetail 获取所有命名空间详细信息
func (dm *DatabaseManager) GetAllNamespacesDetail() ([]Namespace, error) {
	var namespaces []Namespace
	result := dm.db.Find(&namespaces)
	return namespaces, result.Error
}

// DeletePodByEnvironment 根据环境名称删除pod数据
func (dm *DatabaseManager) DeletePodByEnvironment(environment string) (int64, error) {
	result := dm.db.Where("environment = ?", environment).Delete(&Pod{})
	return result.RowsAffected, result.Error
}

// InsertPods 批量插入pod数据
func (dm *DatabaseManager) InsertPods(pods []Pod) error {
	return dm.db.Transaction(func(tx *gorm.DB) error {
		for _, pod := range pods {
			if err := tx.Create(&pod).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetPodCountByEnvironment 根据环境名称获取pod数量
func (dm *DatabaseManager) GetPodCountByEnvironment(environment string) (int64, error) {
	var count int64
	result := dm.db.Model(&Pod{}).Where("environment = ?", environment).Count(&count)
	return count, result.Error
}

// GetPodsByEnvNamespace 根据环境名称和命名空间查询pod列表
func (dm *DatabaseManager) GetPodsByEnvNamespace(environment string, namespaceId string) ([]Pod, error) {
	var pods []Pod
	result := dm.db.Where("environment = ? AND namespace_id = ?", environment, namespaceId).
		Find(&pods)
	return pods, result.Error
}

// GetPodsByEnvNamespaceWorkload 根据环境名称、命名空间和workload查询pod列表
func (dm *DatabaseManager) GetPodsByEnvNamespaceWorkload(environment string, namespaceId string, workload string) ([]Pod, error) {
	var pods []Pod
	result := dm.db.Where("environment = ? AND namespace_id = ? AND workload_id LIKE ?",
		environment, namespaceId, "%"+workload+"%").
		Find(&pods)
	return pods, result.Error
}

// ClearAllData 清除namespace,pod,workload的数据
func (dm *DatabaseManager) ClearAllData() error {
	return dm.db.Transaction(func(tx *gorm.DB) error {
		// 清除namespace数据
		if err := tx.Exec("DELETE FROM namespace").Error; err != nil {
			return err
		}

		// 清除pod数据
		if err := tx.Exec("DELETE FROM pod").Error; err != nil {
			return err
		}

		// 清除workload数据
		if err := tx.Exec("DELETE FROM workload").Error; err != nil {
			return err
		}

		// 清除upload_config数据
		if err := tx.Exec("DELETE FROM upload_config").Error; err != nil {
			return err
		}

		// 清除service数据
		if err := tx.Exec("DELETE FROM service").Error; err != nil {
			return err
		}

		return nil
	})
}

// DeleteAllUploadConfigs 删除所有上传配置数据
func (dm *DatabaseManager) DeleteAllUploadConfigs() error {
	return dm.db.Exec("DELETE FROM upload_config").Error
}

// InsertUploadConfigs 批量插入上传配置数据
func (dm *DatabaseManager) InsertUploadConfigs(configs []UploadConfig) error {
	return dm.db.Transaction(func(tx *gorm.DB) error {
		for _, config := range configs {
			if err := tx.Create(&config).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetUploadConfigsByImage 根据镜像名称精确查询上传配置
func (dm *DatabaseManager) GetUploadConfigsByImage(image string) ([]UploadConfig, error) {
	var configs []UploadConfig
	result := dm.db.Where("image = ?", image).Find(&configs)
	return configs, result.Error
}

// GetUploadConfigsByImageLikeSpecial1 根据镜像名称模糊查询上传配置
func (dm *DatabaseManager) GetUploadConfigsByImageLikeSpecial1(image string) ([]UploadConfig, error) {
	var configs []UploadConfig
	result := dm.db.Where("image LIKE ?", image+":$%").Find(&configs)
	return configs, result.Error
}

// GetUploadConfigsByImageLikeSpecial2 根据镜像名称模糊查询上传配置
func (dm *DatabaseManager) GetUploadConfigsByImageLikeSpecial2(image string) ([]UploadConfig, error) {
	var configs []UploadConfig
	result := dm.db.Where("image LIKE ?", "%/$%/"+image+":$%").Find(&configs)
	return configs, result.Error
}

// GetServicesByWorkload 根据环境、项目ID、命名空间ID和工作负载ID查询服务列表
func (dm *DatabaseManager) GetServicesByWorkload(environment, projectId, namespaceId, workloadId string) ([]Service, error) {
	var services []Service
	result := dm.db.Where("environment = ? AND project_id = ? AND namespace_id = ? AND workload_id = ?",
		environment, projectId, namespaceId, workloadId).Find(&services)
	return services, result.Error
}

// DeleteServiceByEnvironment 根据环境删除服务
func (dm *DatabaseManager) DeleteServiceByEnvironment(environment string) (int64, error) {
	result := dm.db.Where("environment = ?", environment).Delete(&Service{})
	return result.RowsAffected, result.Error
}

// InsertServices 批量插入服务数据
func (dm *DatabaseManager) InsertServices(services []Service) error {
	return dm.db.Transaction(func(tx *gorm.DB) error {
		for _, service := range services {
			if err := tx.Create(&service).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
