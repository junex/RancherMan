package operations

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2/widget"
	"RancherMan/rancher"
)

// jumpHostProgressListener 跳板机进度监听器
type jumpHostProgressListener struct {
	infoArea *widget.Entry
}

// NewJumpHostProgressListener 创建跳板机进度监听器
func NewJumpHostProgressListener(infoArea *widget.Entry) rancher.ProgressListener {
	return &jumpHostProgressListener{
		infoArea: infoArea,
	}
}

func (l *jumpHostProgressListener) OnProgress(currentFolder string, current, total int) {
	// 直接更新 UI
	l.infoArea.SetText(fmt.Sprintf("正在扫描... %d/%d\n当前目录:%s", current, total, currentFolder))
}

func (l *jumpHostProgressListener) OnComplete() {
	// 直接更新 UI
	l.infoArea.SetText("更新跳板机完成")
}

func (l *jumpHostProgressListener) OnBatchResult(configs []rancher.SSHUploadConfig) {
	// 将 SSHUploadConfig 转换为 UploadConfig
	var uploadConfigs []rancher.UploadConfig
	for _, config := range configs {
		uploadConfig := rancher.UploadConfig{
			Dir:    config.Dir,
			Script: config.Script,
			Jar:    config.Jar,
			Image:  config.Image,
		}
		uploadConfigs = append(uploadConfigs, uploadConfig)
	}

	// 插入数据库
	gDb.InsertUploadConfigs(uploadConfigs)
}

// ScanJumpHostTask 创建扫描跳板机配置任务
func ScanJumpHostTask(db *rancher.DatabaseManager, taskQueue *rancher.TaskQueue) {
	newTask := &rancher.Task{
		Type:        rancher.TaskTypeScanJumpHost,
		Description: "扫描跳板机配置",
		DB:          db,
	}
	taskID := taskQueue.AddTask(newTask)
	log.Printf("[ScanJumpHostTask] Task added: ID=%d, Description=%s", taskID, newTask.Description)
	taskQueue.Submit(newTask)
}

// GetJumpHostInfoTask 创建获取目录跳板机信息任务
func GetJumpHostInfoTask(db *rancher.DatabaseManager, taskQueue *rancher.TaskQueue) {
	newTask := &rancher.Task{
		Type:        rancher.TaskTypeGetJumpHostInfo,
		Description: "获取目录跳板机信息",
		DB:          db,
	}
	taskID := taskQueue.AddTask(newTask)
	log.Printf("[GetJumpHostInfoTask] Task added: ID=%d, Description=%s", taskID, newTask.Description)
	taskQueue.Submit(newTask)
}

// UpdateJumpHostDBTask 创建更新跳板机数据库任务
func UpdateJumpHostDBTask(db *rancher.DatabaseManager, taskQueue *rancher.TaskQueue) {
	newTask := &rancher.Task{
		Type:        rancher.TaskTypeUpdateJumpHost,
		Description: "更新跳板机数据库",
		DB:          db,
	}
	taskID := taskQueue.AddTask(newTask)
	log.Printf("[UpdateJumpHostDBTask] Task added: ID=%d, Description=%s", taskID, newTask.Description)
	taskQueue.Submit(newTask)
}
