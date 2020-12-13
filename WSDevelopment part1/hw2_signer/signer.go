package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func parallelCrc32(val string) chan string {
	crcChan := make(chan string, 1)
	go func() {
		defer close(crcChan)
		c := DataSignerCrc32(val)
		crcChan <- c
	}()
	return crcChan
}

type md5Value struct {
	data string
	md5  string
}

func parallelCrcMd5(valToMd5 chan string, mdChanOut chan md5Value) {
	defer close(mdChanOut)
	for {
		if v, ok := <-valToMd5; !ok {
			break
		} else {
			c := md5Value{data: v, md5: DataSignerMd5(v)}
			mdChanOut <- c
		}
	}
}

func concatenateSingle(cm md5Value, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	crc := parallelCrc32(cm.data)
	crcmd := parallelCrc32(cm.md5)
	res := <-crc + "~" + <-crcmd
	out <- res
}

func SingleHash(in, out chan interface{}) {

	md5ChanIn := make(chan string, 10)

	singleWg := &sync.WaitGroup{}
	toCrc := make(chan md5Value, 10)

	go parallelCrcMd5(md5ChanIn, toCrc)

	for {
		signerValRaw, ok := <-in
		if !ok {
			close(md5ChanIn)
			break
		} else {
			signerInt, ok := signerValRaw.(int)
			if !ok {
				return
			}
			signerVal := strconv.Itoa(signerInt)
			md5ChanIn <- signerVal
		}
	}

	for elem := range toCrc {
		singleWg.Add(1)
		go concatenateSingle(elem, out, singleWg)
	}
	singleWg.Wait()
}

func calculateMultiHash(val string, out chan interface{}, wgAll *sync.WaitGroup) {

	var multiHashResult string
	vals := make([]chan string, 6)

	for i := 0; i <= 5; i++ {
		vals[i] = parallelCrc32(strconv.Itoa(i) + val)
	}

	for i := 0; i <= 5; i++ {
		multiHashResult += <-vals[i]
	}

	out <- multiHashResult
	wgAll.Done()
}

func MultiHash(in, out chan interface{}) {

	wg := &sync.WaitGroup{}

	for valRow := range in {
		val, ok := valRow.(string)
		if !ok {
			return
		}
		wg.Add(1)
		go calculateMultiHash(val, out, wg)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {

	dataArr := make([]string, 0)

	for {
		if val, ok := <-in; !ok {
			break
		} else {
			if valString, ok := val.(string); ok {
				dataArr = append(dataArr, valString)
			} else {
				return
			}
		}
	}

	sort.Strings(dataArr)
	combination := strings.Join(dataArr, "_")
	out <- combination
}


func ExecutePipeline(pipelineJobs ...job) {
	in := make(chan interface{})
	out := make(chan interface{})

	mainWg := &sync.WaitGroup{}

	for _, worker := range pipelineJobs {
		nextStep := make(chan interface{})
		mainWg.Add(1)
		go func(in, out chan interface{}, worker func(in, out chan interface{}), wg *sync.WaitGroup) {
			worker(in, out)
			wg.Done()
			close(out)
		}(in, out, worker, mainWg)

		in, out = out, nextStep
	}
	mainWg.Wait()
}

func main() {
	inputData := []int{0, 1, 1, 2, 3, 5, 8, 13, 21, 34}

	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
		job(MultiHash),
		job(CombineResults),
		job(func(in, out chan interface{}) {
			dataRaw := <-in
			data, ok := dataRaw.(string)
			if !ok {
				return
			}
			fmt.Printf("Final result %v\n", data)
		}),
	}

	ExecutePipeline(hashSignJobs...)

}
