// Based on the work at:
// https://eli.thegreenplace.net/2020/pubsub-using-channels-in-go/
//
// PubSub implementation where Subscribe returns a channel.
// Eli Bendersky [https://eli.thegreenplace.net]

package channelpubsub

import (
	"sync"

	exports "github.com/redhat-cne/ptp-listener-exports"
	"github.com/sirupsen/logrus"
)

type Pubsub struct {
	mu     sync.RWMutex
	subs   map[string][]chan exports.StoredEvent
	closed bool
}

func NewPubsub() *Pubsub {
	ps := &Pubsub{}
	ps.subs = make(map[string][]chan exports.StoredEvent)
	ps.closed = false
	return ps
}

func (ps *Pubsub) Subscribe(topic string, channelBuffer int) (chin <-chan exports.StoredEvent, subscriberID int) {
	ps.mu.Lock()
	logrus.Debugf("lock Subscribe %s", topic)
	defer logrus.Debugf("unlock Subscribe %s", topic)
	defer ps.mu.Unlock()

	ch := make(chan exports.StoredEvent, channelBuffer)
	ps.subs[topic] = append(ps.subs[topic], ch)
	return ch, len(ps.subs[topic]) - 1
}

func (ps *Pubsub) Publish(topic string, msg exports.StoredEvent) {
	ps.mu.RLock()
	logrus.Debugf("lock Publish %s", topic)
	defer logrus.Debugf("unlock Publish %s", topic)
	defer ps.mu.RUnlock()

	if ps.closed {
		return
	}

	for _, ch := range ps.subs[topic] {
		if ch == nil {
			continue
		}
		logrus.Debugf("Publish topic=%s channel len=%d, capacity=%d", topic, len(ch), cap(ch))
		ch <- msg
	}
}

func (ps *Pubsub) Close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if !ps.closed {
		ps.closed = true
		for _, subs := range ps.subs {
			for _, ch := range subs {
				if ch == nil {
					continue
				}
				close(ch)
			}
		}
	}
}

func (ps *Pubsub) Unsubscribe(topic string, subscriberID int) {
	ps.mu.Lock()
	logrus.Debug("lock Unsubscribe")
	defer logrus.Debug("unlock Unsubscribe")
	defer ps.mu.Unlock()
	if _, ok := ps.subs[topic]; !ok {
		logrus.Errorf("Unsubscribe: did not find pubsub topic ID=%s", topic)
		return
	}
	if len(ps.subs[topic]) > subscriberID {
		close(ps.subs[topic][subscriberID])
		ps.subs[topic][subscriberID] = nil
	}
}
