package workers

import (
	"log"
	"sync"

	utils "github.com/PabloPavan/rinha2025/utils"
)

type Task func()

type Pool struct {
	tasks chan Task
	wg    sync.WaitGroup
}

func NewPool(numWorkers int) *Pool {
	p := &Pool{
		tasks: make(chan Task, utils.GetEnvInt("POOLSIZE", 10000)),
	}
	p.wg.Add(numWorkers)
	for range numWorkers {
		go p.worker()
	}
	return p
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for task := range p.tasks {
		task()
	}
}

func (p *Pool) Submit(task Task) {
	select {
	case p.tasks <- task:
	default:
		log.Println("WARNING: pool cheio, tarefa descartada ou atrasada")
	}
}

func (p *Pool) Wait() {
	close(p.tasks)
	p.wg.Wait()
}
