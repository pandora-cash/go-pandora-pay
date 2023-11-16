package multicast

import (
	"golang.org/x/exp/slices"
	"pandora-pay/helpers/linked_list"
	"pandora-pay/helpers/safe_channel"
	"sync"
)

type MulticastChannel[T any] struct {
	listeners           []*safe_channel.SafeChannel[T]
	queueBroadcastCn    chan T
	internalBroadcastCn chan T
	count               int
	lock                *sync.RWMutex
}

func (self *MulticastChannel[T]) AddListener() chan T {

	safeCn := safe_channel.NewSafeChannel[T](1)

	self.lock.Lock()
	defer self.lock.Unlock()
	self.listeners = append(self.listeners, safeCn)

	return safeCn.Ch
}

func (self *MulticastChannel[T]) Broadcast(data T) {
	self.queueBroadcastCn <- data
}

func (self *MulticastChannel[T]) RemoveChannel(channel chan T) bool {

	self.lock.Lock()
	defer self.lock.Unlock()

	for i := 0; i < len(self.listeners); i++ {
		if self.listeners[i].Ch == channel {
			close(channel)
			self.listeners = slices.Delete(self.listeners, i, i+1)
			return true
		}
	}

	return false
}

func (self *MulticastChannel[T]) CloseAll() {

	self.lock.Lock()
	defer self.lock.Unlock()

	for _, cn := range self.listeners {
		cn.Close()
	}
	self.listeners = make([]*safe_channel.SafeChannel[T], 0)
	close(self.internalBroadcastCn)
}

func (self *MulticastChannel[T]) runQueueBroadcast() {

	linkedList := linked_list.NewLinkedList[T]()

	for {
		if first, ok := linkedList.GetHead(); ok {
			select {
			case data, ok := <-self.queueBroadcastCn:
				if !ok {
					return
				}
				linkedList.Push(data)
			case self.internalBroadcastCn <- first:
				linkedList.PopHead()
			}
		} else {
			select {
			case data, ok := <-self.queueBroadcastCn:
				if !ok {
					return
				}
				linkedList.Push(data)
			}
		}
	}

}

func (self *MulticastChannel[T]) runInternalBroadcast() {

	for {
		data, ok := <-self.internalBroadcastCn
		if !ok {
			return
		}

		self.lock.RLock()
		listeners := slices.Clone(self.listeners)
		self.lock.RUnlock()

		for _, channel := range listeners {
			channel.Write(data)
		}
	}
}

func NewMulticastChannel[T any]() *MulticastChannel[T] {

	multicast := &MulticastChannel[T]{
		make([]*safe_channel.SafeChannel[T], 0),
		make(chan T),
		make(chan T),
		0,
		&sync.RWMutex{},
	}

	go multicast.runInternalBroadcast()
	go multicast.runQueueBroadcast()

	return multicast
}
