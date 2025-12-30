package operations

import (
	"RancherMan/rancher"
)

// UpdateJumpHostTask 创建更新跳板机配置任务
func UpdateJumpHostTask(db *rancher.DatabaseManager, config *rancher.JumpHostConfig, taskQueue *rancher.TaskQueue) {
	newTask := &rancher.Task{
		Type:           rancher.TaskTypeUpdateJumpHost,
		Description:    "更新跳板机",
		DB:             db,
		JumpHostConfig: config,
	}
	taskQueue.AddTask(newTask)
}
