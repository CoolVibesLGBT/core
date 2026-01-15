package workers

import "sync"

type Dispatcher struct {
	WorkerPool chan TaskQueue
	MaxWorkers int
	TaskQueue  TaskQueue
	wg         *sync.WaitGroup
}

func NewDispatcher(maxWorkers int, queueSize int) *Dispatcher {
	return &Dispatcher{
		WorkerPool: make(chan TaskQueue, maxWorkers),
		MaxWorkers: maxWorkers,
		TaskQueue:  make(TaskQueue, queueSize),
		wg:         &sync.WaitGroup{},
	}
}

func (d *Dispatcher) Run() {
	for i := 0; i < d.MaxWorkers; i++ {
		worker := NewWorker(i, d.WorkerPool, d.wg)
		worker.Start()
	}

	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for task := range d.TaskQueue {
		workerQueue := <-d.WorkerPool
		workerQueue <- task
	}
}

func (d *Dispatcher) SubmitEx(task func()) {
	d.wg.Add(1)         // Bir iş eklendi, bekleme sayısını artır
	d.TaskQueue <- task // TaskQueue'ya task gönder
}

func (d *Dispatcher) Submit(task Task) {
	d.wg.Add(1)
	d.TaskQueue <- task
}

func (d *Dispatcher) Stop() {
	close(d.TaskQueue) // TaskQueue kapatılırsa, dispatch döngüsü sona erer
}

func (d *Dispatcher) Wait() {
	d.wg.Wait()
}
