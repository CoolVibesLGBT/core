package workers

type WorkerPool struct {
	dispatcher *Dispatcher
}

func NewWorkerPool(maxWorkers int) *WorkerPool {
	dispatcher := NewDispatcher(maxWorkers, 1000)
	dispatcher.Run()
	return &WorkerPool{dispatcher: dispatcher}
}

func (wp *WorkerPool) Submit(task Task) {
	wp.dispatcher.wg.Add(1)
	wp.dispatcher.TaskQueue <- task
}

func (wp *WorkerPool) Wait() {
	wp.dispatcher.Wait()
}

func (d *Dispatcher) StopAndWait() {
	d.Wait()
	d.Stop()
}
