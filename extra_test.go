package main

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

/*
	Тест, предложенный одним из учащихся курса, Ilya Boltnev
	https://www.coursera.org/learn/golang-webservices-1/discussions/weeks/2/threads/kI2PR_XtEeeWKRIdN7jcig

	В чем его преимущество по сравнению с TestPipeline?
	1. Он проверяет то, что все функции действительно выполнились
	2. Он дает представление о влиянии time.Sleep в одном из звеньев конвейера на время работы

	возможно кому-то будет легче с ним
	при правильной реализации ваш код конечно же должен его проходить
*/

func TestByIlia(t *testing.T) {

	var recieved uint32
	freeFlowJobs := []job{
		job(func(in, out chan interface{}) {
			out <- uint32(1)
			out <- uint32(3)
			out <- uint32(4)
		}),
		job(func(in, out chan interface{}) {
			for val := range in {
				out <- val.(uint32) * 3
				time.Sleep(time.Millisecond * 100)
			}
		}),
		job(func(in, out chan interface{}) {
			for val := range in {
				fmt.Println("collected ", val)
				atomic.AddUint32(&recieved, val.(uint32))
			}
		}),
	}

	start := time.Now()

	ExecutePipeline(freeFlowJobs...)

	end := time.Since(start)

	expectedTime := time.Millisecond * 350

	if end > expectedTime {
		t.Errorf("execition too long\nGot: %s\nExpected: <%s", end, expectedTime)
	}

	if recieved != (1+3+4)*3 {
		t.Errorf("f3 have not collected inputs, recieved = %d", recieved)
	}
}

func BenchmarkSprintf(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := fmt.Sprintf("%s~%s", "alskdfjsldkf", "alskdfjsldkf")
			//time.Sleep(time.Microsecond * 10)
			_ = result
		}
	})
}

func BenchmarkConcat(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := "aaabbbcccddda" + "~" + "aaabbbcccdddb"
			_ = result
		}
	})
}

func BenchmarkSprintfWV(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			f, s := "aaabbbcccddda", "aaabbbcccdddb"
			result := fmt.Sprintf("%s~%s", f, s)
			_ = result
		}
	})
}

func BenchmarkConcatWV(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			f, s := "aaabbbcccddda", "aaabbbcccdddb"
			result := f + "~" + s
			_ = result
		}
	})
}

func BenchmarkCombineResultsAppendString(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var res string
			f := []string{"aaabbbcccddda", "aaabbbcccdddb", "aaabbbcccdddc", "aaabbbcccdddd", "aaabbbcccddde", "aaabbbcccdddf", "aaabbbcccdddg", "aaabbbcccdddh"}
			for _, v := range f {
				res += "_" + v
			}
			_ = res
		}
	})
}

func BenchmarkCombineResultsStringsJoin(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var res string
			f := []string{"aaabbbcccddda", "aaabbbcccdddb", "aaabbbcccdddc", "aaabbbcccdddd", "aaabbbcccddde", "aaabbbcccdddf", "waaabbbcccdddg", "aaabbbcccdddh"}
			res = strings.Join(f, "_")
			_ = res
		}
	})
}

func BenchmarkCombineResultsMyStringJoin(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var res string
			f := []string{"aaabbbcccddda", "aaabbbcccdddb", "aaabbbcccdddc", "aaabbbcccdddd", "aaabbbcccddde", "aaabbbcccdddf", "waaabbbcccdddg", "aaabbbcccdddh"}

			var buf bytes.Buffer
			lenOfStrings := len(f) - 1
			for i := 0; i < len(f); i++ {
				lenOfStrings += len(f[i])
			}
			buf.Grow(lenOfStrings)

			buf.WriteString(f[0])
			for _, v := range f[1:] {
				buf.WriteByte('_')
				buf.WriteString(v)
			}
			//res = buf.String()
			res = *(*string)(unsafe.Pointer(&(buf.Bytes()[0])))
			_ = res
		}
	})
}

func BenchmarkTestData1(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			wg := sync.WaitGroup{}
			wg.Add(2)
			var crc32Hash, crc32md5Hash string
			go func() {
				crc32md5Hash = "DataSignerCrc32(2432562fg)"
				defer wg.Done()
			}()
			go func() {
				crc32Hash = "DataSignerCrc32(89123567986d8)"
				defer wg.Done()
			}()
			wg.Wait()
			_ = crc32Hash
			_ = crc32md5Hash
		}

	})
}
func BenchmarkTestData3(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			crc32Result := make(chan string)
			crc32md5Result := make(chan string)
			go func() {
				crc32md5Result <- "DataSignerCrc32(2432562fg)"
			}()
			go func() {
				crc32Result <- "DataSignerCrc32(89123567986d8)"
			}()
			crc32Hash := <-crc32Result
			crc32md5Hash := <-crc32md5Result
			_ = crc32Hash
			_ = crc32md5Hash
		}
	})
}
func BenchmarkTestDataSimple2(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			wg := sync.WaitGroup{}
			wg.Add(2)
			result := [2]string{"", ""}
			go func() {
				result[0] = "DataSignerCrc32(2432562fg)"
				wg.Done()
			}()
			go func() {
				result[1] = "DataSignerCrc32(89123567986d8)"
				wg.Done()
			}()
			wg.Wait()
			_ = result[0]
			_ = result[1]
		}
	})
}
