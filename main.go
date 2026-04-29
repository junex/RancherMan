package main

import (
	"log"
	"time"

	"RancherMan/operations"
	"RancherMan/rancher"
	"RancherMan/ui"

	"fyne.io/fyne/v2"
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
	window.SetCloseIntercept(func() {
		taskQueue.Stop()
		window.Close()
	})

	// 加载配置
	operations.LoadConfig(false)

	// 初始化数据
	operations.InitData()

	// 启动任务队列
	taskQueue.Start(&taskQueueUI{})
	time.AfterFunc(300*time.Millisecond, func() {
		db := operations.GetDb()
		taskQueue := operations.GetTaskQueue()
		config := operations.GetConfig()
		for envName, _ := range config["environment"].(map[interface{}]interface{}) {
			environment, _ := rancher.GetEnvironmentFromConfig(config, envName.(string))
			operations.UpdatePodsTask(environment, db, taskQueue)
		}
	})

	window.ShowAndRun()
}

// taskQueueUI 实现任务队列UI回调
type taskQueueUI struct{}

func (t *taskQueueUI) UpdateStatus(status string, dots string, current int, total int, taskInfo string) {
	fyne.Do(func() {
		taskStatusBar := operations.GetTaskStatusBar()
		if taskStatusBar != nil {
			taskStatusBar.Update(status, dots, taskInfo, current, total)
		}

		hasRunning := (status == "执行")
		for _, btn := range operations.GetOperationButtons() {
			if hasRunning {
				btn.Disable()
			} else {
				btn.Enable()
			}
		}
	})
}

func (t *taskQueueUI) OnTaskComplete(task *rancher.Task, result rancher.TaskResult) {
	fyne.DoAndWait(func() {
		log.Printf("[OnTaskComplete] task=%s type = %d success=%v", task.Description, task.Type, result.Success)
		switch task.Type {
		case rancher.TaskTypeUpdateDataComplete, rancher.TaskTypeClearData:
			operations.InitData()
		}
		operations.UpdateInfoArea()
	})
}
