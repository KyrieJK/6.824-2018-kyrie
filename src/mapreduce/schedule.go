package mapreduce

import (
	"fmt"
	"sync"
)

//
// schedule() starts and waits for all tasks in the given phase (mapPhase
// or reducePhase). the mapFiles argument holds the names of the files that
// are the inputs to the map phase, one per map task. nReduce is the
// number of reduce tasks. the registerChan argument yields a stream
// of registered workers; each item is the worker's RPC address,
// suitable for passing to call(). registerChan will yield all
// existing registered workers (if any) and new ones as they register.
//
func schedule(jobName string, mapFiles []string, nReduce int, phase jobPhase, registerChan chan string) {
	var ntasks int
	var n_other int // number of inputs (for reduce) or outputs (for map)
	switch phase {
	case mapPhase:
		ntasks = len(mapFiles) //mapTask任务数=输入文件个人
		n_other = nReduce      //生成的intermediate文件数=Reduce分区数R
	case reducePhase:
		ntasks = nReduce        //reduceTask任务数=Reduce分区数R
		n_other = len(mapFiles) //reduceTask的输入端为mapTask生成的intermediate文件数
	}

	fmt.Printf("Schedule: %v %v tasks (%d I/Os)\n", ntasks, phase, n_other)

	// All ntasks tasks have to be scheduled on workers. Once all tasks
	// have completed successfully, schedule() should return.
	//
	// Your code here (Part III, Part IV).
	//

	wg := sync.WaitGroup{}
	wg.Add(ntasks)
	failedCh := make(chan int, ntasks)

	dispatchTaskToWorker := func(worker string, idx int) {
		args := DoTaskArgs{
			jobName,
			mapFiles[idx],
			phase,
			idx,
			n_other,
		}
		done := call(worker, "Worker.DoTask", args, nil)
		if done {
			wg.Done()
		} else {
			failedCh <- idx
		}
		registerChan <- worker
	}
	for i := 0; i < ntasks; i++ {
		freeWorker := <-registerChan
		go dispatchTaskToWorker(freeWorker, i)
	}

	go func() {
		for {
			idx := <-failedCh
			freeWorker := <-registerChan
			go dispatchTaskToWorker(freeWorker, idx)
		}
	}()
	wg.Wait()

	fmt.Printf("Schedule: %v done\n", phase)
}
