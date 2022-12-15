package p2p

import (
	"sync"
	"time"
)

// MessageIDSubscriber
type MessageIDSubscriber struct {
	lock        *sync.Mutex
	subscribers map[string]chan *Message
}

// NewMessageIDSubscriber create a new instance of MessageIDSubscriber
func NewMessageIDSubscriber() *MessageIDSubscriber {
	return &MessageIDSubscriber{
		lock:        &sync.Mutex{},
		subscribers: make(map[string]chan *Message),
	}
}

// Subscribe a message id
func (ms *MessageIDSubscriber) Subscribe(msgID string, channel chan *Message) {
	ms.lock.Lock()
	defer ms.lock.Unlock()
	ms.subscribers[msgID] = channel
}

// UnSubscribe a messageid
func (ms *MessageIDSubscriber) UnSubscribe(msgID string) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	sb, ok := ms.subscribers[msgID]
	if !ok {
		return
	}
	delete(ms.subscribers, msgID)
	// we clean up the channel as handler in communication.go L180 may be blocked as it need to write to the channel
	for {
		select {
		case <-sb:
		case <-time.After(time.Second):
			return
		}
	}
}

// GetSubscribers return a subscriber of given message id
func (ms *MessageIDSubscriber) GetSubscriber(msgID string) chan *Message {
	ms.lock.Lock()
	defer ms.lock.Unlock()
	c, ok := ms.subscribers[msgID]
	if !ok {
		return nil
	}
	return c
}

// IsEmpty check whether there is subscribers
func (ms *MessageIDSubscriber) IsEmpty() bool {
	ms.lock.Lock()
	defer ms.lock.Unlock()
	return len(ms.subscribers) == 0
}
