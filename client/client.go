package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"bytes"
	"encoding/json"
)

const url = "http://localhost:8082/"


var (
	MaxWorkers = 4
	MaxQueue   = 27
	count      = 0
	id         = 0
)

type Foo struct {
	Key   int `json:"key"`
	Value int    `json:"value"`
}

func (foo Foo) do(client http.Client) {
	var jsonStr, err = json.Marshal(foo)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
}

type Job struct {
	Foo Foo
}

var JobQueue chan Job

type Worker struct {
	WorkerPool chan chan Job
	JobChannel chan Job
	quit       chan bool
	id 		   int
	Client     http.Client
}

func NewWorker(workerPool chan chan Job) Worker {
	id++
	return Worker{
		WorkerPool: workerPool,
		JobChannel: make(chan Job),
		quit:       make(chan bool),
		id: 		id,
		Client:     http.Client{}}
}

func (w Worker) Start() {
	go func() {
		for {
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				job.Foo.do(w.Client)
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

	var rateLimit int
	var countOfRequests int

	fmt.Print("Enter count of worker: ")
	fmt.Scanf("%d", &MaxWorkers)
	fmt.Print("Enter size of queue: ")
	fmt.Scanf("%d", &MaxQueue)

	JobQueue = make(chan Job, MaxQueue)
	fmt.Print("Enter count of requests per minute: ")
	fmt.Scanf("%d", &rateLimit)
	fmt.Print("Enter count of total requests: ")
	fmt.Scanf("%d", &countOfRequests)

	t := float64(60.0 / float64(rateLimit))
	t1 := strconv.FormatFloat(t, 'f', 8, 64) + "s"
	timeout, err := (time.ParseDuration(t1))

	fmt.Println(t1)
	if err != nil {
		panic(err)
	}

	fmt.Println(timeout)

	fmt.Println(time.Now())

	dispatcher := NewDispatcher(MaxWorkers)
	dispatcher.Run()


	for i := 0; i < countOfRequests; i++{
		go func() {
			foo := &Foo{Key: i, Value: i}
			work := Job{Foo: *foo}
			JobQueue <- work

		}()
		time.Sleep(timeout)
	}

	fmt.Println(time.Now())
}

type Dispatcher struct {
	WorkerPool chan chan Job
	maxWorkers int
}

func NewDispatcher(maxWorkers int) *Dispatcher {
	pool := make(chan chan Job, maxWorkers)
	return &Dispatcher{WorkerPool: pool, maxWorkers: maxWorkers}
}

func (d *Dispatcher) Run() {
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewWorker(d.WorkerPool)
		worker.Start()
	}

	go d.dispatch()
}

func (d *Dispatcher) dispatch() {

	for {
		select {
		case job := <-JobQueue:
			go func(job Job) {
				jobChannel := <-d.WorkerPool
				jobChannel <- job
			}(job)
		}
	}
}
