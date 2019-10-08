package trading

import (
	"container/ring"
	"context"
	"sync"

	"time"

	"sync/atomic"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type clusterTrader struct {
	logger    *zap.Logger
	available []Trader
	mx        sync.Mutex
	enabled   *ring.Ring
	assigned  sync.Map
	ready     chan bool
	isReady   uint32
}

func (c *clusterTrader) getGate(userID uint64) (Trader, error) {
	clientInterface, ok := c.assigned.Load(userID)
	if !ok {
		gate := c.getNextEnabled()
		if gate == nil {
			return nil, errors.New("no available trading gates")
		}
		clientInterface, _ = c.assigned.LoadOrStore(userID, gate)
	}
	if client, casting := clientInterface.(Trader); casting {
		return client, nil
	}
	return nil, errors.New("bad implementation get gate")
}

func (c *clusterTrader) Unsubscribe(userID uint64) {
	clientInterface, ok := c.assigned.Load(userID)
	if ok {
		c.assigned.Delete(userID)
		if client, casting := clientInterface.(Trader); casting {
			client.Unsubscribe(userID)
		}
	}
}

func (c *clusterTrader) GetSnapshot(ctx context.Context, userID uint64) ([]OrderModel, error) {
	gate, err := c.getGate(userID)
	if err != nil {
		return nil, err
	}
	return gate.GetSnapshot(ctx, userID)
}

func (c *clusterTrader) GetOrder(ctx context.Context, userID uint64, clientOrderID ClientOrderId) (*OrderModel, error) {
	gate, err := c.getGate(userID)
	if err != nil {
		return nil, err
	}
	return gate.GetOrder(ctx, userID, clientOrderID)
}

func (c *clusterTrader) ReplaceOrder(ctx context.Context, userID uint64, req RequestOrderReplace) (*OrderReport, error) {
	gate, err := c.getGate(userID)
	if err != nil {
		return nil, err
	}
	return gate.ReplaceOrder(ctx, userID, req)
}

func (c *clusterTrader) CancelOrder(ctx context.Context, userID uint64, clientOrderID ClientOrderId) (*OrderReport, error) {
	gate, err := c.getGate(userID)
	if err != nil {
		return nil, err
	}
	return gate.CancelOrder(ctx, userID, clientOrderID)
}

func (c *clusterTrader) NewOrder(ctx context.Context, userID uint64, order *RequestOrderNew) (*OrderReport, []Trade, error) {
	gate, err := c.getGate(userID)
	if err != nil {
		return nil, nil, err
	}
	return gate.NewOrder(ctx, userID, order)
}

func (c *clusterTrader) IsReady() bool {
	return atomic.LoadUint32(&c.isReady) == 1
}

func (c *clusterTrader) setReady(val bool) {
	var state uint32
	if val {
		state = 1
	}

	if atomic.SwapUint32(&c.isReady, state) != state {
		if val {
			c.logger.Info("cluster trading: ready")
		} else {
			c.logger.Error("cluster trading: sadness")
		}
		select {
		case c.ready <- val:
			// ok
		default:
			c.logger.Error("cluster trading: discarding ready call chan capacity")
		}
	}
}

func (c *clusterTrader) Ready() chan bool {
	return c.ready
}

func (c *clusterTrader) getNextEnabled() interface{} {
	c.mx.Lock()
	defer c.mx.Unlock()
	if c.enabled.Len() == 0 {
		return nil
	}
	c.enabled = c.enabled.Next()
	return c.enabled.Value
}

// move each user one by one to prevent degradation latency of core trading gate
func (c *clusterTrader) moveUsersForAvailableGates() {
	c.assigned.Range(func(user, value interface{}) bool {
		userID := user.(uint64)
		if gate, casting := value.(Trader); casting {
			if !gate.IsReady() {
				nextGateInt := c.getNextEnabled()
				if nextGateInt == nil {
					return false
				}
				if nextGate, casting := nextGateInt.(Trader); casting {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
					defer cancel()
					_, err := nextGate.GetSnapshot(ctx, userID)
					if err == nil {
						c.assigned.Store(userID, nextGate)
						gate.Unsubscribe(userID)
					}
				}
			}
		}
		return true
	})
}

// update enabled (ready for serve) list of trading gates
func (c *clusterTrader) updateEnabled() {
	c.mx.Lock()
	enabledCount := 0
	for _, client := range c.available {
		if client.IsReady() {
			enabledCount++
		}
	}
	c.enabled = ring.New(enabledCount)
	for _, client := range c.available {
		if client.IsReady() {
			c.enabled = c.enabled.Next()
			c.enabled.Value = client
		}
	}
	c.mx.Unlock()
	c.setReady(enabledCount > 0)
}

func (c *clusterTrader) handleReady() {
	statusChanged := make(chan interface{}, 10)
	for _, clientVal := range c.available {
		go func(clientObj Trader) {
			for status := range clientObj.Ready() {
				statusChanged <- status
			}
		}(clientVal)
	}

	for range statusChanged {
		c.updateEnabled()

		go c.moveUsersForAvailableGates()
	}
}

func newClusterTrader(logger *zap.Logger, clients []Trader) *clusterTrader {
	cluster := &clusterTrader{
		logger:    logger,
		available: clients,
		ready:     make(chan bool, 2),
	}
	go cluster.handleReady()

	return cluster
}
