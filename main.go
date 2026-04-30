package main

import (
	"log"
	"time"

	"RancherMan/operations"
	"RancherMan/rancher"
	"RancherMan/ui"
	"RancherMan/ui/component"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
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
	result := ui.InitView()
	window := result.Window
	window.SetCloseIntercept(func() {
		taskQueue.Stop()
		window.Close()
	})

	// 加载配置
	operations.LoadConfig(false)

	// 启动任务队列
	taskQueue.Start(&taskQueueUI{
		taskStatusBar:    result.TaskStatusBar,
		operationButtons: result.OperationButtons,
	})
	time.AfterFunc(300*time.Millisecond, func() {
		db := operations.GetDb()
		taskQueue := operations.GetTaskQueue()
		config := operations.GetConfig()
		for envName, _ := range config["environment"].(map[interface{}]interface{}) {
			environment, _ := rancher.GetEnvironmentFromConfig(config, envName.(string))
			operations.UpdatePodsTask(environment, db, taskQueue)
		}
	})

	// 延迟初始化数据，确保在 ShowAndRun 后执行（Fyne 渲染管线就绪）
	time.AfterFunc(30*time.Millisecond, func() {
		operations.InitData()
	})

	window.ShowAndRun()
}

// taskQueueUI 实现任务队列UI回调
type taskQueueUI struct {
	taskStatusBar    *component.TaskStatusBar
	operationButtons []*widget.Button
}

func (t *taskQueueUI) UpdateStatus(status string, dots string, current int, total int, taskInfo string) {
	fyne.Do(func() {
		if t.taskStatusBar != nil {
			t.taskStatusBar.Update(status, dots, taskInfo, current, total)
		}

		hasRunning := (status == "执行")
		for _, btn := range t.operationButtons {
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
