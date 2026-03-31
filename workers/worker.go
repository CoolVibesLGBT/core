package workers

import "sync"

type Worker struct {
	id          int
	WorkerPool  chan TaskQueue
	TaskChannel TaskQueue
	wg          *sync.WaitGroup
}

func NewWorker(id int, workerPool chan TaskQueue, wg *sync.WaitGroup) Worker {
	return Worker{
		id:          id,
		WorkerPool:  workerPool,
		TaskChannel: make(TaskQueue),
		wg:          wg,
	}
}

func (w Worker) Start() {
	go func() {
		for {
			w.WorkerPool <- w.TaskChannel
			task := <-w.TaskChannel
			task()
			w.wg.Done()
		}
	}()
}
