package rancher

import (
	"context"
	"errors"
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
	TaskTypeUpdateData // 新增：更新数据
	TaskTypeUpdateDataComplete
	TaskTypeUpdatePortMap  // 新增：更新端口映射
	TaskTypeUpdateJumpHost // 新增：扫描跳板机配置
	TaskTypeClearData      // 新增：清空数据
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
	ID             int
	Type           TaskType
	Status         TaskStatus
	UnCancelable   bool
	Description    string
	Environment    Environment
	Namespace      string
	Workload       string
	Replicas       int
	Error          error
	DB             *DatabaseManager
	JumpHostConfig *JumpHostConfig

	ctx    context.Context
	cancel context.CancelFunc
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
	tasks           []*Task
	taskCounter     int
	mu              sync.Mutex
	taskChan        chan *Task
	stopChan        chan struct{}
	cancelRequested bool
	isRunning       bool
	currentTask     *Task
	ui              TaskQueueUI
}

// NewTaskQueue 创建新的任务队列
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		tasks:           make([]*Task, 0),
		taskChan:        make(chan *Task, 100),
		stopChan:        make(chan struct{}),
		cancelRequested: false,
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

	// 第一个任务添加时，重置取消标志
	if tq.cancelRequested {
		log.Printf("[AddTask] Resetting cancelRequested flag")
		tq.cancelRequested = false
	}

	tq.taskCounter++
	task.ID = tq.taskCounter
	task.Status = TaskStatusPending

	// 为每个任务创建独立的 context
	task.ctx, task.cancel = context.WithCancel(context.Background())

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

	log.Printf("[CancelAll] Cancelling all tasks")

	// 设置取消标志
	tq.cancelRequested = true

	if tq.currentTask != nil && !tq.currentTask.UnCancelable {
		log.Printf("[CancelAll] Cancelling current running task: ID=%d", tq.currentTask.ID)
		if tq.currentTask.cancel != nil {
			tq.currentTask.cancel()
		}
	}

	// 重建任务列表（只保留不可取消 or 非 Pending）
	newTasks := make([]*Task, 0, len(tq.tasks))
	for _, t := range tq.tasks {
		if t.Status == TaskStatusPending && !t.UnCancelable {
			t.Status = TaskStatusCancelled
			if t.cancel != nil {
				t.cancel()
			}
			continue
		}
		newTasks = append(newTasks, t)
	}

	tq.tasks = newTasks
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

	// UI 更新 goroutine
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		dotStates := []string{".", "..", "...", ".."}
		dotIndex := 0

		for {
			select {
			case <-tq.stopChan:
				return
			case <-ticker.C:
				dotIndex = (dotIndex + 1) % len(dotStates)
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

	// 任务执行 goroutine
	go func() {
		for {
			select {
			case <-tq.stopChan:
				return
			case task := <-tq.taskChan:
				log.Printf("[TaskQueue] Received task from channel: ID=%d, Description=%s", task.ID, task.Description)

				// 检查是否已取消
				tq.mu.Lock()
				cancelled := tq.cancelRequested && !task.UnCancelable
				tq.mu.Unlock()

				if cancelled {
					log.Printf("[TaskQueue] Task skipped (cancelRequested): ID=%d", task.ID)
					tq.mu.Lock()
					task.Status = TaskStatusCancelled
					tq.mu.Unlock()
					continue
				}

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
		success, err := Redeploy(task.ctx, task.Environment, task.Namespace, task.Workload)
		result.Success = success
		if err != nil {
			result.Error = err
		}

	case TaskTypeScaleOpen:
		log.Printf("[executeTask] Executing ScaleOpen for %s", task.Workload)
		success, err := Scale(task.ctx, task.Environment, task.Namespace, task.Workload, 1)
		result.Success = success
		result.Error = err

	case TaskTypeScaleClose:
		log.Printf("[executeTask] Executing ScaleClose for %s", task.Workload)
		success, err := Scale(task.ctx, task.Environment, task.Namespace, task.Workload, 0)
		result.Success = success
		result.Error = err

	case TaskTypeUpdatePod:
		log.Printf("[executeTask] Executing UpdatePod for environment %s", task.Environment.Name)
		err := UpdatePod(task.ctx, task.DB, &task.Environment)
		result.Success = err == nil
		result.Error = err

	case TaskTypeUpdateData:
		log.Printf("[executeTask] Executing UpdateData for environment %s", task.Environment.Name)
		err := UpdateEnvironment(task.ctx, task.DB, &task.Environment, true)
		result.Success = err == nil
		result.Error = err

	case TaskTypeUpdateDataComplete:
		log.Printf("[executeTask] Executing UpdateDataComplete")
		result.Success = true

	case TaskTypeUpdatePortMap:
		log.Printf("[executeTask] Executing UpdatePortMap for environment %s", task.Environment.Name)
		err := UpdateService(task.ctx, task.DB, &task.Environment)
		result.Success = err == nil
		result.Error = err

	case TaskTypeUpdateJumpHost:
		log.Printf("[executeTask] Executing UpdateJumpHost")
		success, err := UpdateJumpHostConfig(task.ctx, task.DB, task)
		result.Success = success
		result.Error = err

	case TaskTypeClearData:
		log.Printf("[executeTask] Executing ClearData")
		err := task.DB.ClearAllData()
		result.Success = err == nil
		result.Error = err
	}

	tq.mu.Lock()
	// 检查是否是因为取消而结束
	if errors.Is(task.ctx.Err(), context.Canceled) {
		task.Status = TaskStatusCancelled
		result.Error = fmt.Errorf("任务已取消")
		result.Success = false
	} else if result.Success {
		task.Status = TaskStatusCompleted
	} else {
		task.Status = TaskStatusFailed
		task.Error = result.Error
	}

	tq.currentTask = nil
	allDone := !tq.hasUnfinishedTasks()

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

func (tq *TaskQueue) Stop() {
	close(tq.stopChan)
	log.Printf("[TaskQueue.Stop] Task queue stopped")
}
