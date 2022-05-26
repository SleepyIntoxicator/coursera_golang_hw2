package messageBroker

import (
	"errors"
	"sync"
)

type subscribeItem struct {
	subscribersCount    int
	cachedResultPromise chan interface{}
	resultValue         interface{}
}
type IMessageBroker interface {
	TrySubscribe(inputValue interface{}) (<-chan interface{}, bool)
	StoreResultByInput(inputValue, resultValue interface{}) error
}

type MessageBroker struct {
	subsList sync.Map
}

func NewMessageBroker() IMessageBroker {
	return &MessageBroker{}
}

// TrySubscribe returns non-nil chan and true if one of the workers is already calculating
// the result for this input value; otherwise, it returns an empty chan and false
func (b *MessageBroker) TrySubscribe(inputValue interface{}) (<-chan interface{}, bool) {
	// TODO: WARNING: SOLVED: may cause deadlock if try to subscribe after giveaways results
	//var subsItem subscribeItem

	//t := time.Now()
	existingSubsItem, isValueInCache := b.subsList.Load(inputValue)

	// If one of the workers is processing calculations or already have a ready result
	// then copy subscribe item (because we can't modify elements in the map)
	if isValueInCache {
		existingSubsItem := existingSubsItem.(subscribeItem)

		// If the expected result value is already in the cache then immediately returns the filled chan
		if existingSubsItem.resultValue != nil {
			existingSubsItem.cachedResultPromise <- existingSubsItem.resultValue
			//fmt.Printf("Eated time: %s\n", time.Since(t))
			return existingSubsItem.cachedResultPromise, true
		}

		subsItem := subscribeItem{
			subscribersCount:    existingSubsItem.subscribersCount + 1,
			cachedResultPromise: existingSubsItem.cachedResultPromise,
			resultValue:         existingSubsItem.resultValue,
		}

		b.subsList.Store(inputValue, subsItem)

		//fmt.Printf("Eated time: %s\n", time.Since(t))
		return subsItem.cachedResultPromise, true
	}

	// If none of the workers processes the calculations,
	// add a new subscription item with a key (input value) to subscribe list and return false (calculations needed)
	subsItem := subscribeItem{
		subscribersCount:    0,
		cachedResultPromise: make(chan interface{}, 1),
		resultValue:         nil,
	}
	b.subsList.Store(inputValue, subsItem)

	return subsItem.cachedResultPromise, false
}

// StoreResultByInput sets the value for a key
func (b *MessageBroker) StoreResultByInput(inputValue, resultValue interface{}) error {
	//TODO: ( NO ) delete cache after sending all results
	//t := time.Now()

	read, valueCached := b.subsList.Load(inputValue)
	if !valueCached {
		return errors.New("unexpected input")
	}
	subsItem := read.(subscribeItem)

	err := sendResultsToWaitingSubscribers(resultValue, &subsItem.subscribersCount, subsItem.cachedResultPromise)
	if err != nil {
		return err
	}

	subsItem.resultValue = resultValue
	b.subsList.Store(inputValue, subsItem)

	//fmt.Printf("Eated time: %s\n", time.Since(t))
	return nil
}

func sendResultsToWaitingSubscribers(resultValue interface{}, subscribersCount *int, sendChan chan interface{}) error {
	if *subscribersCount < 0 {
		return errors.New("negative value of subscribers")
	}
	for i := 0; i < *subscribersCount; i++ {
		sendChan <- resultValue
		*subscribersCount--
	}
	return nil
}
