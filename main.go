package main

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// сюда писать код

type handler func(input interface{}, mu *sync.Mutex) string

const (
	isDebug  = false
	useCache = false
)

// runDistributedInputHandler goes through each value received in the input channel and
// launches one distributed handler for each value.
// Closes when the input channel becomes closed and all handlers in goroutines are executed.
// Accepts 2 chans (input, output) and a handler which returns a string result.
func runDistributedInputHandler(in, out chan interface{}, handler handler) {
	//A single mutex for synchronizing one operation (md5) in each calculation.
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	MBroker := NewMessageBroker()

	for input := range in {

		if useCache {
			resChan, waitForResult := MBroker.TrySubscribe(input)
			if isDebug {
				fmt.Printf("run: [SUBSCRIBE]\t[ input: %-24v ]\t[ wait for result: %v ]\n", input, waitForResult)
			}
			if waitForResult {
				debug := <-resChan
				if isDebug {
					fmt.Printf("run: [CACHE GOT]\t[ from cache: %-24v ]\n", debug)
				}

				out <- debug
				continue
			}
		}

		wg.Add(1)

		go func(input interface{}, mustCache bool) {
			defer wg.Done()

			if mustCache {
				result := handler(input, &mu)

				if err := MBroker.StoreResultByInput(input, result); err != nil {
					log.Println("Error: ", err)
				}

				out <- result
			} else {
				out <- handler(input, &mu)
			}

		}(input, useCache)

	}
	wg.Wait()
}

func calculateCrc32(data string, dst *string, wg *sync.WaitGroup) {
	defer wg.Done()
	*dst = DataSignerCrc32(data)
}

func SingleHash(in, out chan interface{}) {
	hasher := handler(func(input interface{}, mu *sync.Mutex) string {
		inputNum, ok := input.(int)
		if !ok {
			panic(fmt.Sprint("input data type is not a number", input))
		}

		inputStr := strconv.Itoa(inputNum)

		mu.Lock()
		md5Hash := DataSignerMd5(inputStr)
		mu.Unlock()

		wg := sync.WaitGroup{}
		wg.Add(2)
		var crc32Hash, crc32md5Hash string

		go calculateCrc32(inputStr, &crc32Hash, &wg)
		go calculateCrc32(md5Hash, &crc32md5Hash, &wg)

		wg.Wait()

		result := crc32Hash + "~" + crc32md5Hash

		if isDebug {
			fmt.Println(input, "SingleHash data", input)
			fmt.Println(input, "SingleHash md5(data)", md5Hash)
			fmt.Println(input, "SingleHash crc32(md5(data))", crc32md5Hash)
			fmt.Println(input, "SingleHash crc32(data)", crc32Hash)
			fmt.Println(input, "SingleHash result:", result)
		}

		return result
	})

	runDistributedInputHandler(in, out, hasher)
}

func MultiHash(in, out chan interface{}) {
	hasher := handler(func(input interface{}, mu *sync.Mutex) string {
		if _, ok := input.(string); !ok {
			panic(fmt.Sprint("input data type is not a string", input))
		}

		phases := 6
		distributeComputingResult := make([]string, phases)

		wg := sync.WaitGroup{}
		wg.Add(phases)
		for th := 0; th < phases; th++ {
			th := th
			go calculateCrc32(
				strconv.Itoa(th)+input.(string),
				&distributeComputingResult[th],
				&wg)
		}
		wg.Wait()

		result := strings.Join(distributeComputingResult, "")
		if isDebug {
			for th := 0; th < phases; th++ {
				fmt.Printf("%s MultiHash: crc32(th+step1) %d %s\n", input, th, distributeComputingResult[th])
			}

			fmt.Printf("%s MultiHash result: %s\n\n", input, result)
		}

		return result
	})

	runDistributedInputHandler(in, out, hasher)
}

func CombineResults(in, out chan interface{}) {
	var results []string

	for input := range in {
		results = append(results, input.(string))
	}

	if isDebug {
		log.Println(" <|°_°|> COMBINE RESULTS COUNT:", len(results))
	}

	sort.Strings(results)
	out <- strings.Join(results, "_")
}

func Worker(job job, in, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(out)
	//t := time.Now()

	job(in, out)

	//fmt.Printf("Worker done with time: %s\n", time.Since(t))
}

func ExecutePipeline(jobs ...job) {
	fmt.Println("Start pipeline...")

	wg := sync.WaitGroup{}

	var in chan interface{}
	for i := range jobs {
		wg.Add(1)
		out := make(chan interface{}, 1)

		go Worker(jobs[i], in, out, &wg)

		in = out
	}
	wg.Wait()
}

func main() {
	MyTest()
}

// MyTest
// benchmark results without using cache and random numbers:	  500 items -> 9.86 s
// benchmark results without using cache and repeatable numbers:  500 items -> 9.9 s
//
// benchmark results using the cache and random numbers:	 	  500 items -> 7.19 s
// benchmark results using the cache and repeatable numbers:	  500 items -> 3.13 s
func MyTest() {
	inputData := []int{0, 1, 1, 2, 3, 5, 8}
	rand.Seed(time.Now().Unix())

	// 500 items
	for i := 0; i < 493; i++ {
		inputData = append(inputData, rand.Intn(10))
	}

	//for i := 0; i < 493; i++ {
	//	inputData = append(inputData, 5)
	//}

	jobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
		job(MultiHash),
		job(CombineResults),
		job(func(in, out chan interface{}) {
			fmt.Printf("CombineResults %s\n", <-in)
		}),
	}

	start := time.Now()
	ExecutePipeline(jobs...)
	end := time.Since(start)

	fmt.Printf("execition time\nGot: %s\nExpected: <%s\n", end, time.Second*7)
}
