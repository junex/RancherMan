package rancher

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// TaskType 定义任务类型
type TaskType int

const (
	TaskTypeRedeploy TaskType = iota
	TaskTypeScaleOpen
	TaskTypeScaleClose
	TaskTypeUpdatePod
	TaskTypeUpdateData      // 新增：更新数据
	TaskTypeUpdatePortMap   // 新增：更新端口映射
	TaskTypeScanJumpHost    // 新增：扫描跳板机配置
	TaskTypeGetJumpHostInfo // 新增：获取目录跳板机信息
	TaskTypeUpdateJumpHost  // 新增：更新跳板机数据库
	TaskTypeClearData       // 新增：清空数据
)

// TaskStatus 定义任务状态
type TaskStatus int

const (
	TaskStatusPending TaskStatus = iota
	TaskStatusRunning
	TaskStatusCompleted
	TaskStatusFailed
	TaskStatusCancelled
)

// Task 任务结构体
type Task struct {
	ID          int
	Type        TaskType
	Status      TaskStatus
	Description string
	Environment Environment
	Namespace   string
	Workload    string
	Replicas    int
	Error       error
	DB          *DatabaseManager // 添加数据库引用
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID  int
	Success bool
	Error   error
}

// TaskQueueUI 任务队列UI更新回调接口
type TaskQueueUI interface {
	UpdateStatus(status string, dots string, current int, total int, taskInfo string)
	OnTaskComplete(task *Task, result TaskResult)
}

// TaskQueue 任务队列管理器
type TaskQueue struct {
	tasks       []*Task
	taskCounter int
	mu          sync.Mutex
	taskChan    chan *Task
	cancelChan  chan struct{}
	isRunning   bool
	currentTask *Task
	ui          TaskQueueUI
}

// NewTaskQueue 创建新的任务队列
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		tasks:      make([]*Task, 0),
		taskChan:   make(chan *Task, 100),
		cancelChan: make(chan struct{}),
	}
}

// SetUI 设置UI回调
func (tq *TaskQueue) SetUI(ui TaskQueueUI) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	tq.ui = ui
}

// AddTask 添加任务到队列
func (tq *TaskQueue) AddTask(task *Task) int {
	log.Printf("[AddTask] ENTRY: Trying to acquire lock...")
	tq.mu.Lock()
	log.Printf("[AddTask] Lock acquired!")
	defer tq.mu.Unlock()

	tq.taskCounter++
	task.ID = tq.taskCounter
	task.Status = TaskStatusPending
	tq.tasks = append(tq.tasks, task)

	log.Printf("[AddTask] Task ID=%d, Type=%d, Description=%s", task.ID, task.Type, task.Description)

	return task.ID
}

// Submit 提交任务到执行队列
func (tq *TaskQueue) Submit(task *Task) {
	tq.taskChan <- task
}

// CancelAll 取消所有任务
func (tq *TaskQueue) CancelAll() {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// 标记所有待执行任务为已取消
	for _, t := range tq.tasks {
		if t.Status == TaskStatusPending {
			t.Status = TaskStatusCancelled
		}
	}

	// 发送取消信号
	select {
	case tq.cancelChan <- struct{}{}:
	default:
	}

	// 清空任务列表
	tq.tasks = make([]*Task, 0)
}

// GetTaskCount 获取任务总数和当前执行序号
func (tq *TaskQueue) GetTaskCount() (current, total int) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	total = len(tq.tasks)

	completedCount := 0
	for _, t := range tq.tasks {
		if t.Status == TaskStatusCompleted || t.Status == TaskStatusFailed {
			completedCount++
		}
	}

	// 如果有正在执行的任务，current = 已完成数 + 1
	// 否则 current = 已完成数
	if tq.currentTask != nil && (tq.currentTask.Status == TaskStatusRunning) {
		current = completedCount + 1
	} else {
		current = completedCount
	}

	return current, total
}

// GetCurrentTask 获取当前正在执行的任务
func (tq *TaskQueue) GetCurrentTask() *Task {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	return tq.currentTask
}

// HasRunningTasks 检查是否有正在运行或待运行的任务
func (tq *TaskQueue) HasRunningTasks() bool {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	for _, t := range tq.tasks {
		if t.Status == TaskStatusRunning || t.Status == TaskStatusPending {
			return true
		}
	}
	return false
}

// Start 启动任务执行器
func (tq *TaskQueue) Start(ui TaskQueueUI) {
	tq.SetUI(ui)
	log.Printf("[TaskQueue.Start] Task queue started")

	// 启动ticker用于更新UI
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		dotStates := []string{".", "..", "...", ".."}
		dotIndex := 0

		for {
			select {
			case <-tq.cancelChan:
				return

			case <-ticker.C:
				dotIndex = (dotIndex + 1) % len(dotStates)

				// 不在这里获取锁，让内部方法自己处理
				current, total := tq.GetTaskCount()
				hasRunning := tq.HasRunningTasks()

				var taskInfo string
				if hasRunning {
					tq.mu.Lock()
					currentTask := tq.currentTask
					tq.mu.Unlock()

					if currentTask != nil {
						taskInfo = currentTask.Description
					}
					tq.ui.UpdateStatus("执行", dotStates[dotIndex], current, total, taskInfo)
				} else {
					tq.ui.UpdateStatus("空闲", "", 0, 0, "")
				}
			}
		}
	}()

	// 启动任务执行goroutine
	go func() {
		for {
			select {
			case <-tq.cancelChan:
				return
			case task := <-tq.taskChan:
				log.Printf("[TaskQueue] Received task from channel: ID=%d, Description=%s", task.ID, task.Description)
				tq.executeTask(task)
			}
		}
	}()
}

// executeTask 执行单个任务
func (tq *TaskQueue) executeTask(task *Task) {
	log.Printf("[executeTask] Starting task: ID=%d, Description=%s", task.ID, task.Description)

	tq.mu.Lock()
	tq.currentTask = task
	task.Status = TaskStatusRunning
	tq.mu.Unlock()

	var result TaskResult
	result.TaskID = task.ID

	// 根据任务类型执行相应操作
	switch task.Type {
	case TaskTypeRedeploy:
		log.Printf("[executeTask] Executing Redeploy for %s", task.Workload)
		success := Redeploy(task.Environment, task.Namespace, task.Workload)
		result.Success = success
		if !success {
			result.Error = fmt.Errorf("重新部署失败")
		}

	case TaskTypeScaleOpen:
		log.Printf("[executeTask] Executing ScaleOpen for %s", task.Workload)
		success := Scale(task.Environment, task.Namespace, task.Workload, 1)
		result.Success = success

	case TaskTypeScaleClose:
		log.Printf("[executeTask] Executing ScaleClose for %s", task.Workload)
		success := Scale(task.Environment, task.Namespace, task.Workload, 0)
		result.Success = success

	case TaskTypeUpdatePod:
		log.Printf("[executeTask] Executing UpdatePod for environment %s", task.Environment.Name)
		UpdatePod(task.DB, task.Environment.ID, &task.Environment)
		result.Success = true

	case TaskTypeUpdateData:
		log.Printf("[executeTask] Executing UpdateData for environment %s", task.Environment.Name)
		result.Success = UpdateEnvironmentData(task.DB, task.Environment.ID, &task.Environment, tq.ui)

	case TaskTypeUpdatePortMap:
		log.Printf("[executeTask] Executing UpdatePortMap for environment %s", task.Environment.Name)
		result.Success = UpdateServiceData(task.DB, task.Environment.ID, &task.Environment, tq.ui)

	case TaskTypeScanJumpHost:
		log.Printf("[executeTask] Executing ScanJumpHost")
		result.Success = ScanJumpHostConfig(task.DB, &task.Environment)

	case TaskTypeGetJumpHostInfo:
		log.Printf("[executeTask] Executing GetJumpHostInfo")
		result.Success = GetJumpHostDirectoryInfo(task.DB, &task.Environment)

	case TaskTypeUpdateJumpHost:
		log.Printf("[executeTask] Executing UpdateJumpHost")
		result.Success = UpdateJumpHostDatabase(task.DB)

	case TaskTypeClearData:
		log.Printf("[executeTask] Executing ClearData")
		err := task.DB.ClearAllData()
		result.Success = err == nil
		if err != nil {
			result.Error = err
		}
	}

	tq.mu.Lock()
	if result.Success {
		task.Status = TaskStatusCompleted
	} else {
		task.Status = TaskStatusFailed
		task.Error = result.Error
	}
	tq.currentTask = nil
	// 1. 判断是否全部完成
	allDone := !tq.hasUnfinishedTasks()

	// 2. 如果全部完成，清空队列
	if allDone {
		tq.tasks = make([]*Task, 0)
	}
	tq.mu.Unlock()

	log.Printf("[executeTask] Task completed: ID=%d, Success=%v", task.ID, result.Success)
	tq.ui.OnTaskComplete(task, result)
}

func (tq *TaskQueue) hasUnfinishedTasks() bool {
	for _, t := range tq.tasks {
		if t.Status == TaskStatusPending || t.Status == TaskStatusRunning {
			return true
		}
	}
	return false
}
