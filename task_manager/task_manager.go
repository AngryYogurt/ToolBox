package task_manager

import (
	"sync"
	"time"
)

type TaskParam = interface{}
type TaskResult struct {
	Result interface{}
	Err    error
}

type Task struct {
	Params  []*TaskParam
	Handle  func(param *TaskParam) (result *TaskResult)
	Results []*TaskResult
}

type TaskManager struct {
	Interval    time.Duration
	Task        *Task
	WorkerCount int
}

func NewTask(params []*TaskParam, handle func(p *TaskParam) *TaskResult) *Task {
	return &Task{
		Params:  params,
		Handle:  handle,
		Results: make([]*TaskResult, 0),
	}
}

func NewTaskManager(interval time.Duration, task *Task, workerCount int) *TaskManager {
	return &TaskManager{
		Interval:    interval,
		Task:        task,
		WorkerCount: workerCount,
	}
}

func (t *TaskManager) Start() *sync.WaitGroup {
	c := make(chan int, len(t.Task.Params)+1)
	for i, _ := range t.Task.Params {
		c <- i
	}
	c <- -1
	wg := &sync.WaitGroup{}
	for i := 0; i < t.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			p := <-c
			for p >= 0 {
				time.Sleep(t.Interval)
				t.Task.Results = append(t.Task.Results, t.Task.Handle(t.Task.Params[p]))
				p = <-c
			}
			c <- -1
			wg.Done()
		}()
		time.Sleep(5*time.Second)
	}
	return wg
}

func (t *TaskManager) GetTaskResult() map[*TaskParam]*TaskResult {
	res := map[*TaskParam]*TaskResult{}
	for i, _ := range t.Task.Params {
		res[t.Task.Params[i]] = t.Task.Results[i]
	}
	return res
}
