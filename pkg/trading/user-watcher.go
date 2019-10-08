package trading

import (
	"sync"
	"time"
)

type UserWatcher struct {
	mx       sync.Mutex
	users    map[uint64]int64
	duration time.Duration
	trader   Trader
}

func NewUserWatcher(trader Trader, duration time.Duration) *UserWatcher {
	watcher := &UserWatcher{
		users:    make(map[uint64]int64),
		duration: duration,
		trader:   trader,
	}
	go func() {
		for range time.NewTicker(duration).C {
			watcher.release()
		}
	}()
	return watcher
}

func (w *UserWatcher) Touch(userID uint64) {
	w.mx.Lock()
	w.users[userID] = time.Now().Unix()
	w.mx.Unlock()
}

func (w *UserWatcher) release() {
	obsoleteTime := time.Now().Unix() - int64(w.duration.Seconds())
	w.mx.Lock()
	for userID, timestamp := range w.users {
		if timestamp < obsoleteTime {
			w.trader.Unsubscribe(userID)
			delete(w.users, userID)
		}
	}
	w.mx.Unlock()
}
