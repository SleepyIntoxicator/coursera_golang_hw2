package messageBroker

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"
)

func TestMessageBroker_StoreResultByInput(t *testing.T) {
	inputValue := 123
	resultValue := 1337

	broker := NewMessageBroker()

	// Set the value for caching
	nilOut, cannotWait := broker.TrySubscribe(inputValue)
	fmt.Printf("nilOut = %v | cannotWait = %v\n", nilOut, cannotWait)

	// Success subscribe
	out, canWait := broker.TrySubscribe(inputValue)
	fmt.Printf("Out = %v | cannotWait = %v\n", out, canWait)
	if canWait {
		go func(out <-chan interface{}) {
			fmt.Printf("Start waiting result...\n")
			// Waiting the result
			resOut := <-out
			fmt.Printf("Result gotted: %d\n", resOut.(int))

			if resOut != resultValue {
				t.Errorf("error: result value is not equal input value")
				t.Errorf("got: %d", inputValue)
				t.Errorf("expected: %d", resultValue)
				t.Fail()
			}
		}(out)
	}
	time.Sleep(2 * time.Second)
	err := broker.StoreResultByInput(inputValue, resultValue)
	if err != nil {
		log.Println(err)
	}
}

func TestSyncMap(t *testing.T) {
	value := 1337
	newValue := 256

	var syncMap sync.Map
	syncMap.Store(1, value)
	syncMap.Store(1, newValue)

	got, _ := syncMap.Load(1)
	if got != newValue {
		t.Error("sync.Map does not replace the value", got)
		t.Fail()
	}
}
