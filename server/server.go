package main

import (
	"log"
	"net/http"
	"encoding/json"
	"fmt"
	"sync/atomic"
)

var (
	MaxWorkers = 0
	MaxQueue   = 0
	count      uint64
	id         = 0
)

type Foo struct {
	Key   int `json:"key"`
	Value int    `json:"value"`
}

type Job struct {
	Foo Foo
}

func (f *Foo) do() {
	log.Println(f, count)
	atomic.AddUint64(&count, 1)
}

func fooHandler(rw http.ResponseWriter, req *http.Request) {

	decoder := json.NewDecoder(req.Body)
	var foo Foo
	err := decoder.Decode(&foo)
	if err != nil {
		panic(err)
	}
	defer req.Body.Close()

	go func(foo Foo) {
		work := Job{foo}

		JobQueue <- work
	}(foo)

}

var JobQueue chan Job

type Worker struct {
	WorkerPool chan chan Job
	JobChannel chan Job
	quit       chan bool
	id 		   int
}

func NewWorker(workerPool chan chan Job) Worker {
	id++
	return Worker{
		WorkerPool: workerPool,
		JobChannel: make(chan Job),
		quit:       make(chan bool),
		id: 		id}
}

func (w Worker) Start() {
	go func() {
		for {
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				go job.Foo.do()
			case <-w.quit:
				return
			}
		}
	}()
}

func (w Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}

func main() {
	fmt.Print("Enter count of worker: ")
	fmt.Scanf("%d", &MaxWorkers)
	fmt.Print("Enter size of queue: ")
	fmt.Scanf("%d", &MaxQueue)
	
	JobQueue = make(chan Job, MaxQueue)

	log.Println("main start")

	dispatcher := NewDispatcher(MaxWorkers)
	dispatcher.Run()

	http.HandleFunc("/", fooHandler)

	err := http.ListenAndServe(":8082", nil)
	if err != nil {
		fmt.Println("starting listening for payload messages")
	} else {
		fmt.Errorf("an error occured while starting payload server %s", err.Error())
	}
}


