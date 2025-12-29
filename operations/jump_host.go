package operations

import (
	"log"

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
	taskID := taskQueue.AddTask(newTask)
	log.Printf("[UpdateJumpHostTask] Task added: ID=%d, Description=%s", taskID, newTask.Description)
	taskQueue.Submit(newTask)
}
