package main

import (
	"log"

	"RancherMan/operations"
	"RancherMan/rancher"
	"RancherMan/ui"
)

func main() {
	// 创建数据库管理器实例
	database, err := rancher.NewDatabaseManager("")
	if err != nil {
		log.Fatal(err)
	}
	operations.SetDb(database)
	defer database.Close()

	// 初始化任务队列
	taskQueue := rancher.NewTaskQueue()
	operations.SetTaskQueue(taskQueue)

	// 初始化UI
	window := ui.InitView()

	// 加载配置
	operations.LoadConfig(false)

	// 初始化数据
	operations.InitData()

	// 启动任务队列
	taskQueue.Start(&taskQueueUI{})

	window.ShowAndRun()
}

// taskQueueUI 实现任务队列UI回调
type taskQueueUI struct{}

func (t *taskQueueUI) UpdateStatus(status string, dots string, current int, total int, taskInfo string) {
	// 使用 goroutine UI 更新确保在主线程中更新
	taskStatusBar := operations.GetTaskStatusBar()
	if taskStatusBar != nil {
		taskStatusBar.Update(status, dots, taskInfo, current, total)
	}

	// 根据是否有任务禁用/启用按钮
	hasRunning := (status == "执行")
	for _, btn := range operations.GetOperationButtons() {
		if hasRunning {
			btn.Disable()
		} else {
			btn.Enable()
		}
	}
}

func (t *taskQueueUI) OnTaskComplete(task *rancher.Task, result rancher.TaskResult) {
	log.Printf("[OnTaskComplete] task=%s type = %i success=%v", task.Description, task.Type, result.Success)
	switch task.Type {
	case rancher.TaskTypeUpdateData, rancher.TaskTypeClearData:
		operations.InitData()
	}
	operations.UpdateInfoArea()
}
