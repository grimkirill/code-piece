package trading

import (
	"sync/atomic"
	"testing"

	"sync"

	"time"

	"context"

	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gotest.tools/assert"
)

// zmq push mock

type mockSendRequestNew func(order payloadOrderNew) error
type mockSendRequestCancel func(order requestOrderCancel) error
type mockSendRequestReplace func(order requestOrderReplacePayload) error

type pushMock struct {
	logger       *zap.Logger
	ready        chan bool
	isReady      uint32
	xsub         *xsubMock
	mocksNew     []mockSendRequestNew
	mocksCancel  []mockSendRequestCancel
	mocksReplace []mockSendRequestReplace
	canaryOn     bool
}

func (c *pushMock) addReplaceMock(requestReplace mockSendRequestReplace) {
	c.mocksReplace = append(c.mocksReplace, requestReplace)
}

func (c *pushMock) addNewMock(requestNew mockSendRequestNew) {
	c.mocksNew = append(c.mocksNew, requestNew)
}

func (c *pushMock) addCancelMock(requestCancel mockSendRequestCancel) {
	c.mocksCancel = append(c.mocksCancel, requestCancel)
}

func (c *pushMock) cleanMocks() {
	c.mocksNew = make([]mockSendRequestNew, 0)
	c.mocksCancel = make([]mockSendRequestCancel, 0)
	c.mocksReplace = make([]mockSendRequestReplace, 0)
}

func (c *pushMock) isMocksEmpty() bool {
	return len(c.mocksNew) == 0 &&
		len(c.mocksCancel) == 0 &&
		len(c.mocksReplace) == 0
}

func (c *pushMock) Ready() chan bool {
	return c.ready
}

func (c *pushMock) SendRequestNew(order payloadOrderNew) error {
	if c.canaryOn && messageUserID(canaryUser) == order.UserID {
		rejectData := &payloadReject{
			userID:        canaryUser,
			ClientOrderID: order.ClientOrderID,
			Timestamp:     uint64(time.Now().Unix() * 1000),
			RejectReason:  orderRejectExceedsLimit,
		}
		c.xsub.reports <- rejectData
		return nil
	}
	c.logger.Info("zmq: mock request new order", zap.Reflect("order", order))
	if len(c.mocksNew) == 0 {
		return errors.New("mock new requests is empty")
	}
	h := c.mocksNew[0]
	c.mocksNew = c.mocksNew[1:]
	return h(order)
}

func (c *pushMock) SendRequestCancel(order requestOrderCancel) error {
	c.logger.Info("zmq: mock request cancel order", zap.Reflect("order", order))
	if len(c.mocksCancel) == 0 {
		return errors.New("mock cancel requests is empty")
	}
	h := c.mocksCancel[0]
	c.mocksCancel = c.mocksCancel[1:]
	return h(order)
}

func (c *pushMock) SendRequestReplace(order requestOrderReplacePayload) error {
	c.logger.Info("zmq: mock request replace order", zap.Reflect("order", order))
	if len(c.mocksReplace) == 0 {
		return errors.New("mock replace requests is empty")
	}
	h := c.mocksReplace[0]
	c.mocksReplace = c.mocksReplace[1:]
	return h(order)
}

func (c *pushMock) IsReady() bool {
	return atomic.LoadUint32(&c.isReady) == 1
}

func (c *pushMock) Close() error {
	return nil
}

func (c *pushMock) setReady(val bool) {
	var state uint32
	if val {
		state = 1
	}

	if atomic.SwapUint32(&c.isReady, state) != state {
		c.logger.Info("zmq: mock push ready", zap.Bool("ready", val))
		select {
		case c.ready <- val:
			// ok
		default:
			panic("zmq: discarding ready call chan capacity")
		}
	}
}

type mockSubscribeHandler func(reports chan interface{}) error

type xsubMock struct {
	addr                 string
	reports              chan interface{}
	logger               *zap.Logger
	ready                chan bool
	isReady              uint32
	mocksSubscribeHandle map[uint64]mockSubscribeHandler
}

func (c *xsubMock) addSubscribeOrdersHandler(userID uint64, h mockSubscribeHandler) {
	if _, ok := c.mocksSubscribeHandle[userID]; ok {
		panic("duplicate snapshot handler")
	}
	c.mocksSubscribeHandle[userID] = h
}

func (c *xsubMock) addSubscribeOrders(userID uint64, orders []OrderModel) {
	if _, ok := c.mocksSubscribeHandle[userID]; ok {
		panic("duplicate snapshot handler")
	}
	c.mocksSubscribeHandle[userID] = func(reports chan interface{}) error {
		reports <- &payloadSnapshot{userID: userID, orders: orders}
		return nil
	}
}

func (c *xsubMock) addSubscribeOrdersTimeout(userID uint64) {
	if _, ok := c.mocksSubscribeHandle[userID]; ok {
		panic("duplicate snapshot handler")
	}
	c.mocksSubscribeHandle[userID] = func(reports chan interface{}) error {
		return nil
	}
}

func (c *xsubMock) cleanMocks() {
	c.mocksSubscribeHandle = make(map[uint64]mockSubscribeHandler)
}

func (c *xsubMock) isMocksEmpty() bool {
	return len(c.mocksSubscribeHandle) == 0
}

func (c *xsubMock) Reports() chan interface{} {
	return c.reports
}

func (c *xsubMock) Ready() chan bool {
	return c.ready
}

// GetAddr get connection endpoint address
func (c *xsubMock) GetAddr() string {
	return c.addr
}

// IsReady return connection ready state.
// If connection established and heartbeat is received, then true
func (c *xsubMock) IsReady() bool {
	return atomic.LoadUint32(&c.isReady) == 1
}

func (c *xsubMock) Close() error {
	return nil
}

// Subscribe allow subscribe for reports for user.
func (c *xsubMock) Subscribe(userID uint64) error {
	c.logger.Info("zmq: subscribe", zap.String("addr", c.addr), zap.Uint64("UserID", userID))
	h, ok := c.mocksSubscribeHandle[userID]
	if !ok {
		c.logger.Error("unexpected subscribe", zap.Uint64("user", userID))
		return errors.New("unexpected subscribe")
	}
	delete(c.mocksSubscribeHandle, userID)
	return h(c.reports)
}

// UnSubscribe allow unsubscribe for reports for user
func (c *xsubMock) UnSubscribe(userID uint64) error {
	c.logger.Info("zmq: unsubscribe", zap.String("addr", c.addr), zap.Uint64("UserID", userID))
	return nil
}

func (c *xsubMock) setReady(val bool) {
	var state uint32
	if val {
		state = 1
	}

	if atomic.SwapUint32(&c.isReady, state) != state {
		c.logger.Info("zmq: mock xsub ready", zap.Bool("ready", val))
		select {
		case c.ready <- val:
			// ok
		default:
			panic("zmq: discarding ready call chan capacity")
		}
	}
}

func createSweetPair(ready bool) (*xsubMock, *pushMock, *zmqTrader) {
	logger, _ := zap.NewDevelopment()
	xsub := &xsubMock{
		logger:               logger,
		ready:                make(chan bool, 1),
		mocksSubscribeHandle: make(map[uint64]mockSubscribeHandler),
		reports:              make(chan interface{}, 100),
	}
	push := &pushMock{
		logger: logger,
		ready:  make(chan bool, 1),
		xsub:   xsub,
	}

	if ready {
		xsub.addSubscribeOrders(canaryUser, make([]OrderModel, 0))
		push.setReady(true)
		xsub.setReady(true)
		push.canaryOn = true
	}
	trader := StartNewClient(logger, push, xsub)
	return xsub, push, trader
}

func TestZmqTrader_Ready(t *testing.T) {

	xsub, push, trader := createSweetPair(false)

	assert.Check(t, !trader.IsReady(), "ready state")

	xsub.addSubscribeOrders(canaryUser, make([]OrderModel, 0))
	push.addNewMock(func(order payloadOrderNew) error {
		assert.Equal(t, messageUserID(canaryUser), order.UserID)
		rejectData := &payloadReject{
			userID:        canaryUser,
			ClientOrderID: order.ClientOrderID,
			Timestamp:     uint64(time.Now().Unix() * 1000),
			RejectReason:  orderRejectExceedsLimit,
		}
		xsub.reports <- rejectData
		return nil
	})
	push.setReady(true)
	xsub.setReady(true)
	assert.Check(t, !trader.IsReady(), "ready state")

	<-trader.Ready()
	assert.Check(t, trader.IsReady(), "ready state")

	push.mocksNew = append(push.mocksNew, func(order payloadOrderNew) error {
		assert.Equal(t, messageUserID(canaryUser), order.UserID)
		return errors.New("some timeout")
	})

	<-trader.Ready()
	assert.Check(t, !trader.IsReady(), "ready state")
	assert.Check(t, push.isMocksEmpty(), "push mock full filled")
	assert.Check(t, xsub.isMocksEmpty(), "xsub mock full filled")
}

var nextUserID uint64 = 123456780

func genUserID() uint64 {
	return atomic.AddUint64(&nextUserID, 1)
}

func genTimestamp() uint64 {
	return uint64(time.Now().Unix() * 1000)
}

func TestZmqTrader_Unsubscribe(t *testing.T) {
	xsub, push, trader := createSweetPair(true)
	<-trader.Ready()
	assert.Check(t, trader.IsReady(), "ready state")

	t.Run("unsubscribe empty", func(t *testing.T) {
		userID := genUserID()

		trader.Unsubscribe(userID)
		assert.Check(t, !trader.orders.hasOrders(userID))
	})

	t.Run("unsubscribe non empty", func(t *testing.T) {
		userID := genUserID()

		xsub.addSubscribeOrdersHandler(userID, func(reports chan interface{}) error {
			reports <- &payloadSnapshot{userID: userID, orders: make([]OrderModel, 0)}
			go func() {
				time.Sleep(time.Millisecond * 10)
				reports <- &payloadSnapshot{userID: userID, orders: make([]OrderModel, 0)}
			}()
			return nil
		})

		orders, err := trader.GetSnapshot(context.Background(), userID)
		assert.NilError(t, err)
		assert.Equal(t, len(orders), 0)
		assert.Check(t, xsub.isMocksEmpty(), "xsub mock full filled")
		trader.Unsubscribe(userID)
		assert.Check(t, !trader.orders.hasOrders(userID))

		time.Sleep(time.Millisecond * 20)
		assert.Check(t, !trader.orders.hasOrders(userID))

		xsub.addSubscribeOrders(userID, make([]OrderModel, 0))
		_, err = trader.GetSnapshot(context.Background(), userID)
		assert.NilError(t, err)
	})

	assert.Check(t, push.isMocksEmpty(), "push mock full filled")
	assert.Check(t, xsub.isMocksEmpty(), "xsub mock full filled")
}

func TestZmqTrader_GetSnapshot(t *testing.T) {
	xsub, push, trader := createSweetPair(true)
	<-trader.Ready()
	assert.Check(t, trader.IsReady(), "ready state")
	t.Run("empty snapshot", func(t *testing.T) {
		userID := genUserID()
		xsub.addSubscribeOrders(userID, make([]OrderModel, 0))

		orders, err := trader.GetSnapshot(context.Background(), userID)
		assert.NilError(t, err)
		assert.Equal(t, len(orders), 0)
		_, err = trader.GetOrder(context.Background(), userID, ClientOrderIdGenerateFast(42))
		assert.ErrorContains(t, err, "orderNotFound")
	})

	t.Run("parallel empty snapshot", func(t *testing.T) {
		userID := genUserID()
		xsub.addSubscribeOrdersHandler(userID, func(reports chan interface{}) error {
			time.Sleep(time.Millisecond * 100)
			reports <- &payloadSnapshot{userID: userID, orders: make([]OrderModel, 0)}
			return nil
		})
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			orders, err := trader.GetSnapshot(context.Background(), userID)
			assert.NilError(t, err)
			assert.Equal(t, len(orders), 0)
		}()
		orders, err := trader.GetSnapshot(context.Background(), userID)
		assert.NilError(t, err)
		assert.Equal(t, len(orders), 0)
		wg.Wait()
	})

	t.Run("parallel empty snapshot with one timeout", func(t *testing.T) {
		userID := genUserID()
		xsub.addSubscribeOrdersHandler(userID, func(reports chan interface{}) error {
			fmt.Println("CHAN", reports)
			time.Sleep(time.Millisecond * 100)

			reports <- &payloadSnapshot{userID: userID, orders: make([]OrderModel, 0)}
			return nil
		})

		var wg sync.WaitGroup
		wg.Add(1)
		ctxSecond, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		go func() {
			defer wg.Done()
			orders, err := trader.GetSnapshot(ctxSecond, userID)
			assert.NilError(t, err)
			assert.Equal(t, len(orders), 0)
		}()

		ctx, cancelMs := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancelMs()
		_, err := trader.GetSnapshot(ctx, userID)
		assert.ErrorContains(t, err, "context deadline exceeded")

		wg.Wait()
		orders, err := trader.GetSnapshot(ctxSecond, userID)
		assert.NilError(t, err)
		assert.Equal(t, len(orders), 0)
	})

	t.Run("filled snapshot", func(t *testing.T) {
		userID := genUserID()
		xsub.addSubscribeOrders(userID, []OrderModel{
			OrderModel{
				TimeInForce:   OrderTimeInForceGTC,
				Type:          OrderTypeLimit,
				Price:         "123",
				Quantity:      "50",
				Symbol:        "BTCUSD",
				PostOnly:      false,
				Side:          OrderSideSell,
				ClientOrderID: ClientOrderIdGenerateFast(42),
				Created:       1565701539017,
				OrderID:       130173734978,
				OrderStatus:   OrderStatusNew,
			},
		})
		order, err := trader.GetOrder(context.Background(), userID, ClientOrderIdGenerateFast(42))
		assert.NilError(t, err)
		assert.Equal(t, order.OrderStatus, OrderStatusNew)
		assert.Equal(t, order.Symbol, "BTCUSD")

		orders, err := trader.GetSnapshot(context.Background(), userID)
		assert.NilError(t, err)
		assert.Equal(t, len(orders), 1)
	})

	t.Run("error snapshot", func(t *testing.T) {
		userID := genUserID()
		xsub.addSubscribeOrdersHandler(userID, func(reports chan interface{}) error {
			return errors.New("expected timeout")
		})
		_, err := trader.GetSnapshot(context.Background(), userID)
		assert.ErrorContains(t, err, "time")
	})

	t.Run("timeout snapshot", func(t *testing.T) {
		userID := genUserID()
		xsub.addSubscribeOrdersTimeout(userID)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := trader.GetSnapshot(ctx, userID)
		assert.ErrorContains(t, err, "context deadline exceeded")
	})

	assert.Check(t, push.isMocksEmpty(), "push mock full filled")
	assert.Check(t, xsub.isMocksEmpty(), "xsub mock full filled")
}

func TestZmqTrader_GetOrder(t *testing.T) {
	xsub, push, trader := createSweetPair(true)
	<-trader.Ready()
	assert.Check(t, trader.IsReady(), "ready state")

	t.Run("timeout get order", func(t *testing.T) {
		userID := genUserID()
		xsub.addSubscribeOrdersTimeout(userID)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := trader.GetOrder(ctx, userID, ClientOrderIdGenerateFast(42))
		assert.ErrorContains(t, err, "context deadline exceeded")
	})

	assert.Check(t, push.isMocksEmpty(), "push mock full filled")
	assert.Check(t, xsub.isMocksEmpty(), "xsub mock full filled")
}

func TestZmqTrader_NewOrder(t *testing.T) {
	xsub, push, trader := createSweetPair(true)
	<-trader.Ready()
	assert.Check(t, trader.IsReady(), "ready state")
	cleanMocks := func() {
		push.cleanMocks()
		xsub.cleanMocks()
	}
	t.Run("timeout place order, timeout snapshot", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		xsub.addSubscribeOrdersTimeout(userID)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _, err := trader.NewOrder(ctx, userID, &RequestOrderNew{
			ClientOrderID: ClientOrderIdGenerateFast(42),
		})
		assert.ErrorContains(t, err, "context deadline exceeded")
	})

	t.Run("timeout place order, timeout response", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		xsub.addSubscribeOrders(userID, make([]OrderModel, 0))
		push.addNewMock(func(order payloadOrderNew) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			return nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _, err := trader.NewOrder(ctx, userID, &RequestOrderNew{
			ClientOrderID: ClientOrderIdGenerateFast(42),
		})
		assert.ErrorContains(t, err, "context deadline exceeded")
	})

	t.Run("place normal order from empty snapshot", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(42)
		xsub.addSubscribeOrders(userID, make([]OrderModel, 0))
		push.addNewMock(func(order payloadOrderNew) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)

			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: OrderModel{
					TimeInForce:   OrderTimeInForceGTC,
					Type:          OrderTypeLimit,
					Price:         "123",
					Quantity:      "50",
					Symbol:        "BTCUSD",
					PostOnly:      false,
					Side:          OrderSideSell,
					ClientOrderID: ClientOrderIdGenerateFast(42),
					Created:       genTimestamp(),
					OrderID:       130173734978,
					OrderStatus:   OrderStatusNew,
				},
				ReportType: ReportTypeNew,
				Timestamp:  genTimestamp(),
			}}

			return nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _, err := trader.NewOrder(ctx, userID, &RequestOrderNew{
			ClientOrderID: clientID,
			Symbol:        "BTCUSD",
			Price:         "1",
			Quantity:      "1",
		})
		assert.NilError(t, err)
	})

	t.Run("place normal order from empty snapshot", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(42)
		xsub.addSubscribeOrders(userID, make([]OrderModel, 0))
		push.addNewMock(func(order payloadOrderNew) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			go func() {
				xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
					OrderModel: OrderModel{
						TimeInForce:   OrderTimeInForceGTC,
						Type:          OrderTypeLimit,
						Price:         "123",
						Quantity:      "50",
						Symbol:        "BTCUSD",
						PostOnly:      false,
						Side:          OrderSideSell,
						ClientOrderID: ClientOrderIdGenerateFast(42),
						Created:       genTimestamp(),
						OrderID:       130173734978,
						OrderStatus:   OrderStatusNew,
					},
					ReportType: ReportTypeNew,
					Timestamp:  genTimestamp(),
				}}
				xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
					OrderModel: OrderModel{
						TimeInForce:   OrderTimeInForceGTC,
						Type:          OrderTypeLimit,
						Price:         "123",
						Quantity:      "50",
						Symbol:        "BTCUSD",
						PostOnly:      false,
						Side:          OrderSideSell,
						ClientOrderID: ClientOrderIdGenerateFast(42),
						Created:       genTimestamp(),
						OrderID:       130173734978,
						OrderStatus:   OrderStatusPartiallyFilled,
					},
					ReportType: ReportTypeTrade,
					Timestamp:  genTimestamp(),
					TradeID:    1234,
				}}
				xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
					OrderModel: OrderModel{
						TimeInForce:   OrderTimeInForceGTC,
						Type:          OrderTypeLimit,
						Price:         "123",
						Quantity:      "50",
						Symbol:        "BTCUSD",
						PostOnly:      false,
						Side:          OrderSideSell,
						ClientOrderID: ClientOrderIdGenerateFast(42),
						Created:       genTimestamp(),
						OrderID:       130173734978,
						OrderStatus:   OrderStatusFilled,
					},
					ReportType: ReportTypeTrade,
					Timestamp:  genTimestamp(),
					TradeID:    1235,
				}}
			}()
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		res, trades, err := trader.NewOrder(ctx, userID, &RequestOrderNew{
			ClientOrderID: clientID,
			Symbol:        "BTCUSD",
			Price:         "1",
			Quantity:      "1",
		})
		assert.NilError(t, err)
		//todo how it works, and why?
		assert.Equal(t, OrderStatusFilled, res.OrderStatus)
		assert.Equal(t, 2, len(trades))
	})

	t.Run("place with duplicate from container", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(42)
		xsub.addSubscribeOrders(userID, []OrderModel{{
			TimeInForce:   OrderTimeInForceGTC,
			Type:          OrderTypeLimit,
			Price:         "123",
			Quantity:      "50",
			Symbol:        "BTCUSD",
			PostOnly:      false,
			Side:          OrderSideSell,
			ClientOrderID: ClientOrderIdGenerateFast(42),
			Created:       genTimestamp(),
			OrderID:       130173734978,
			OrderStatus:   OrderStatusNew,
		},
		})

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _, err := trader.NewOrder(ctx, userID, &RequestOrderNew{
			ClientOrderID: clientID,
			Symbol:        "BTCUSD",
			Price:         "1",
			Quantity:      "1",
		})
		assert.ErrorContains(t, err, "duplicate")
	})

	t.Run("place duplicate pending", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(42)
		xsub.addSubscribeOrders(userID, make([]OrderModel, 0))
		push.addNewMock(func(order payloadOrderNew) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)

			time.Sleep(time.Millisecond * 100)

			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: OrderModel{
					TimeInForce:   OrderTimeInForceGTC,
					Type:          OrderTypeLimit,
					Price:         "123",
					Quantity:      "50",
					Symbol:        "BTCUSD",
					PostOnly:      false,
					Side:          OrderSideSell,
					ClientOrderID: ClientOrderIdGenerateFast(42),
					Created:       genTimestamp(),
					OrderID:       130173734978,
					OrderStatus:   OrderStatusNew,
				},
				ReportType: ReportTypeNew,
				Timestamp:  genTimestamp(),
			}}

			return nil
		})
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _, err := trader.NewOrder(context.Background(), userID, &RequestOrderNew{
				ClientOrderID: clientID,
				Symbol:        "BTCUSD",
				Price:         "1",
				Quantity:      "1",
			})
			assert.NilError(t, err)
		}()

		go func() {
			defer wg.Done()
			time.Sleep(time.Millisecond)
			_, _, err := trader.NewOrder(context.Background(), userID, &RequestOrderNew{
				ClientOrderID: clientID,
				Symbol:        "BTCUSD",
				Price:         "1",
				Quantity:      "1",
			})
			assert.ErrorContains(t, err, "duplicate")
		}()

		wg.Wait()
	})

	t.Run("place FOK expired order from empty snapshot", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(42)
		xsub.addSubscribeOrders(userID, make([]OrderModel, 0))
		push.addNewMock(func(order payloadOrderNew) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			assert.Equal(t, OrderTimeInForceFOK, order.TimeInForce)

			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: OrderModel{
					TimeInForce:   OrderTimeInForceFOK,
					Type:          OrderTypeLimit,
					Price:         "123",
					Quantity:      "50",
					Symbol:        "BTCUSD",
					PostOnly:      false,
					Side:          OrderSideSell,
					ClientOrderID: ClientOrderIdGenerateFast(42),
					Created:       genTimestamp(),
					OrderID:       130173734978,
					OrderStatus:   OrderStatusNew,
				},
				ReportType: ReportTypeNew,
				Timestamp:  genTimestamp(),
			}}
			time.Sleep(time.Millisecond)

			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: OrderModel{
					TimeInForce:   OrderTimeInForceFOK,
					Type:          OrderTypeLimit,
					Price:         "123",
					Quantity:      "50",
					Symbol:        "BTCUSD",
					PostOnly:      false,
					Side:          OrderSideSell,
					ClientOrderID: ClientOrderIdGenerateFast(42),
					Created:       genTimestamp(),
					OrderID:       130173734978,
					OrderStatus:   OrderStatusExpired,
				},
				ReportType: ReportTypeExpired,
				Timestamp:  genTimestamp(),
			}}

			return nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		order, _, err := trader.NewOrder(ctx, userID, &RequestOrderNew{
			ClientOrderID: clientID,
			TimeInForce:   OrderTimeInForceFOK,
			Symbol:        "BTCUSD",
			Price:         "1",
			Quantity:      "1",
			WaitComplete:  true,
		})
		assert.NilError(t, err)
		assert.Equal(t, clientID, order.ClientOrderID)
		assert.Equal(t, OrderStatusExpired, order.OrderStatus)
		assert.Equal(t, OrderTimeInForceFOK, order.TimeInForce)
	})

	t.Run("place FOK traded order from empty snapshot", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(42)
		xsub.addSubscribeOrders(userID, make([]OrderModel, 0))
		push.addNewMock(func(order payloadOrderNew) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			assert.Equal(t, OrderTimeInForceFOK, order.TimeInForce)

			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: OrderModel{
					TimeInForce:   OrderTimeInForceFOK,
					Type:          OrderTypeLimit,
					Price:         "123",
					Quantity:      "50",
					Symbol:        "BTCUSD",
					PostOnly:      false,
					Side:          OrderSideSell,
					ClientOrderID: ClientOrderIdGenerateFast(42),
					Created:       genTimestamp(),
					OrderID:       130173734978,
					OrderStatus:   OrderStatusNew,
				},
				ReportType: ReportTypeNew,
				Timestamp:  genTimestamp(),
			}}
			time.Sleep(time.Millisecond)

			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: OrderModel{
					TimeInForce:   OrderTimeInForceFOK,
					Type:          OrderTypeLimit,
					Price:         "123",
					Quantity:      "50",
					Symbol:        "BTCUSD",
					PostOnly:      false,
					Side:          OrderSideSell,
					ClientOrderID: ClientOrderIdGenerateFast(42),
					Created:       genTimestamp(),
					OrderID:       130173734978,
					OrderStatus:   OrderStatusFilled,
					LastQuantity:  50,
					LastPrice:     "123",
				},
				ReportType: ReportTypeTrade,
				Timestamp:  genTimestamp(),
				TradeID:    1123,
				FeeAmount:  "0.01",
			}}

			return nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		order, trades, err := trader.NewOrder(ctx, userID, &RequestOrderNew{
			ClientOrderID: clientID,
			TimeInForce:   OrderTimeInForceFOK,
			Symbol:        "BTCUSD",
			Price:         "1",
			Quantity:      "1",
			WaitComplete:  true,
		})
		assert.NilError(t, err)
		assert.Equal(t, clientID, order.ClientOrderID)
		assert.Equal(t, OrderStatusFilled, order.OrderStatus)
		assert.Equal(t, OrderTimeInForceFOK, order.TimeInForce)
		assert.Equal(t, 1, len(trades))
		assert.Equal(t, uint64(1123), trades[0].ID)
	})

	t.Run("place IOC expired order from empty snapshot", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(42)
		xsub.addSubscribeOrders(userID, make([]OrderModel, 0))
		push.addNewMock(func(order payloadOrderNew) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			assert.Equal(t, OrderTimeInForceIOC, order.TimeInForce)

			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: OrderModel{
					TimeInForce:   OrderTimeInForceIOC,
					Type:          OrderTypeLimit,
					Price:         "123",
					Quantity:      "50",
					Symbol:        "BTCUSD",
					PostOnly:      false,
					Side:          OrderSideSell,
					ClientOrderID: ClientOrderIdGenerateFast(42),
					Created:       genTimestamp(),
					OrderID:       130173734978,
					OrderStatus:   OrderStatusNew,
				},
				ReportType: ReportTypeNew,
				Timestamp:  genTimestamp(),
			}}
			time.Sleep(time.Millisecond)

			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: OrderModel{
					TimeInForce:   OrderTimeInForceIOC,
					Type:          OrderTypeLimit,
					Price:         "123",
					Quantity:      "50",
					Symbol:        "BTCUSD",
					PostOnly:      false,
					Side:          OrderSideSell,
					ClientOrderID: ClientOrderIdGenerateFast(42),
					Created:       genTimestamp(),
					OrderID:       130173734978,
					OrderStatus:   OrderStatusExpired,
				},
				ReportType: ReportTypeExpired,
				Timestamp:  genTimestamp(),
			}}

			return nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		order, _, err := trader.NewOrder(ctx, userID, &RequestOrderNew{
			ClientOrderID: clientID,
			TimeInForce:   OrderTimeInForceIOC,
			Symbol:        "BTCUSD",
			Price:         "1",
			Quantity:      "1",
			WaitComplete:  true,
		})
		assert.NilError(t, err)
		assert.Equal(t, clientID, order.ClientOrderID)
		assert.Equal(t, OrderStatusExpired, order.OrderStatus)
		assert.Equal(t, OrderTimeInForceIOC, order.TimeInForce)
	})

	t.Run("place IOC traded order from empty snapshot", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(42)
		xsub.addSubscribeOrders(userID, make([]OrderModel, 0))
		push.addNewMock(func(order payloadOrderNew) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			assert.Equal(t, OrderTimeInForceIOC, order.TimeInForce)

			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: OrderModel{
					TimeInForce:   OrderTimeInForceIOC,
					Type:          OrderTypeLimit,
					Price:         "123",
					Quantity:      "50",
					Symbol:        "BTCUSD",
					PostOnly:      false,
					Side:          OrderSideSell,
					ClientOrderID: ClientOrderIdGenerateFast(42),
					Created:       genTimestamp(),
					OrderID:       130173734978,
					OrderStatus:   OrderStatusNew,
				},
				ReportType: ReportTypeNew,
				Timestamp:  genTimestamp(),
			}}
			time.Sleep(time.Millisecond)

			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: OrderModel{
					TimeInForce:   OrderTimeInForceIOC,
					Type:          OrderTypeLimit,
					Price:         "123",
					Quantity:      "50",
					Symbol:        "BTCUSD",
					PostOnly:      false,
					Side:          OrderSideSell,
					ClientOrderID: ClientOrderIdGenerateFast(42),
					Created:       genTimestamp(),
					OrderID:       130173734978,
					OrderStatus:   OrderStatusFilled,
					LastQuantity:  50,
					LastPrice:     "123",
				},
				ReportType: ReportTypeTrade,
				Timestamp:  genTimestamp(),
				TradeID:    1123,
				FeeAmount:  "0.01",
			}}

			return nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		order, trades, err := trader.NewOrder(ctx, userID, &RequestOrderNew{
			ClientOrderID: clientID,
			TimeInForce:   OrderTimeInForceIOC,
			Symbol:        "BTCUSD",
			Price:         "1",
			Quantity:      "1",
			WaitComplete:  true,
		})
		assert.NilError(t, err)
		assert.Equal(t, clientID, order.ClientOrderID)
		assert.Equal(t, OrderStatusFilled, order.OrderStatus)
		assert.Equal(t, OrderTimeInForceIOC, order.TimeInForce)
		assert.Equal(t, 1, len(trades))
		assert.Equal(t, uint64(1123), trades[0].ID)
	})

	assert.Check(t, push.isMocksEmpty(), "push mock full filled")
	assert.Check(t, xsub.isMocksEmpty(), "xsub mock full filled")
	assert.Check(t, len(trader.pendingFilled) == 0, "new calls is empty")
}

func TestZmqTrader_CancelOrder(t *testing.T) {
	xsub, push, trader := createSweetPair(true)
	<-trader.Ready()
	assert.Check(t, trader.IsReady(), "ready state")
	cleanMocks := func() {
		push.cleanMocks()
		xsub.cleanMocks()
	}
	t.Run("reject order not found in container", func(t *testing.T) {

		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(22)
		xsub.addSubscribeOrders(userID, make([]OrderModel, 0))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := trader.CancelOrder(ctx, userID, clientID)
		assert.Equal(t, ErrorNotFoundOrder, err)
	})

	t.Run("timeout cancel order, timeout snapshot", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		xsub.addSubscribeOrdersTimeout(userID)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		_, err := trader.CancelOrder(ctx, userID, ClientOrderIdGenerateFast(42))
		assert.ErrorContains(t, err, "context deadline exceeded")
	})

	t.Run("cancel order from container with canceled", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(62)
		orderSnap := OrderModel{
			TimeInForce:   OrderTimeInForceGTC,
			Type:          OrderTypeLimit,
			Price:         "123",
			Quantity:      "50",
			Symbol:        "BTCUSD",
			PostOnly:      false,
			Side:          OrderSideSell,
			ClientOrderID: clientID,
			Created:       genTimestamp(),
			OrderID:       130173734978,
			OrderStatus:   OrderStatusNew,
		}
		xsub.addSubscribeOrders(userID, []OrderModel{orderSnap})

		push.addCancelMock(func(order requestOrderCancel) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			orderCancel := orderSnap
			orderCancel.OrderStatus = OrderStatusCanceled
			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: orderCancel,
				ReportType: ReportTypeCanceled,
				Timestamp:  genTimestamp(),
			}}
			return nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		report, err := trader.CancelOrder(ctx, userID, clientID)
		assert.NilError(t, err)
		assert.Equal(t, OrderStatusCanceled, report.OrderStatus)
	})

	t.Run("timeout cancel order, timeout response", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(62)
		orderSnap := OrderModel{
			TimeInForce:   OrderTimeInForceGTC,
			Type:          OrderTypeLimit,
			Price:         "123",
			Quantity:      "50",
			Symbol:        "BTCUSD",
			PostOnly:      false,
			Side:          OrderSideSell,
			ClientOrderID: clientID,
			Created:       genTimestamp(),
			OrderID:       130173734978,
			OrderStatus:   OrderStatusNew,
		}
		xsub.addSubscribeOrders(userID, []OrderModel{orderSnap})

		push.addCancelMock(func(order requestOrderCancel) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			// skip publish report as timeout
			return nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		_, err := trader.CancelOrder(ctx, userID, clientID)
		assert.ErrorContains(t, err, "context deadline exceeded")
	})

	t.Run("fail cancel order, fail response", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(62)
		orderSnap := OrderModel{
			TimeInForce:   OrderTimeInForceGTC,
			Type:          OrderTypeLimit,
			Price:         "123",
			Quantity:      "50",
			Symbol:        "BTCUSD",
			PostOnly:      false,
			Side:          OrderSideSell,
			ClientOrderID: clientID,
			Created:       genTimestamp(),
			OrderID:       130173734978,
			OrderStatus:   OrderStatusNew,
		}
		xsub.addSubscribeOrders(userID, []OrderModel{orderSnap})

		push.addCancelMock(func(order requestOrderCancel) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			// skip publish report as timeout
			return errors.New("zmq error")
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		_, err := trader.CancelOrder(ctx, userID, clientID)
		assert.ErrorContains(t, err, "zmq error")
	})

	t.Run("cancel order from container with filled", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(62)
		orderSnap := OrderModel{
			TimeInForce:   OrderTimeInForceGTC,
			Type:          OrderTypeLimit,
			Price:         "123",
			Quantity:      "50",
			Symbol:        "BTCUSD",
			PostOnly:      false,
			Side:          OrderSideSell,
			ClientOrderID: clientID,
			Created:       genTimestamp(),
			OrderID:       130173734978,
			OrderStatus:   OrderStatusNew,
		}
		xsub.addSubscribeOrders(userID, []OrderModel{orderSnap})

		push.addCancelMock(func(order requestOrderCancel) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			orderCancel := orderSnap
			orderCancel.OrderStatus = OrderStatusFilled
			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: orderCancel,
				ReportType: ReportTypeTrade,
				Timestamp:  genTimestamp(),
			}}
			return nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		report, err := trader.CancelOrder(ctx, userID, clientID)
		assert.NilError(t, err)
		assert.Equal(t, OrderStatusFilled, report.OrderStatus)
	})

	t.Run("cancel order from container with reject", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(62)
		orderSnap := OrderModel{
			TimeInForce:   OrderTimeInForceGTC,
			Type:          OrderTypeLimit,
			Price:         "123",
			Quantity:      "50",
			Symbol:        "BTCUSD",
			PostOnly:      false,
			Side:          OrderSideSell,
			ClientOrderID: clientID,
			Created:       genTimestamp(),
			OrderID:       130173734978,
			OrderStatus:   OrderStatusNew,
		}
		xsub.addSubscribeOrders(userID, []OrderModel{orderSnap})

		push.addCancelMock(func(order requestOrderCancel) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			orderCancel := orderSnap
			orderCancel.OrderStatus = OrderStatusFilled
			xsub.reports <- &payloadReject{
				userID:                     userID,
				ClientOrderID:              clientID,
				CancelRequestClientOrderID: order.CancelRequestClientOrderID,
				RejectReason:               orderRejectNotFound,
			}
			return nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := trader.CancelOrder(ctx, userID, clientID)
		assert.ErrorContains(t, err, "orderNotFound")
	})

	t.Run("cancel pending new order from empty snapshot with filled", func(t *testing.T) {
		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(42)
		xsub.addSubscribeOrders(userID, make([]OrderModel, 0))

		push.addCancelMock(func(order requestOrderCancel) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			return nil
		})
		push.addNewMock(func(order payloadOrderNew) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			assert.Equal(t, OrderTimeInForceGTC, order.TimeInForce)

			time.Sleep(time.Millisecond * 100)
			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: OrderModel{
					TimeInForce:   OrderTimeInForceGTC,
					Type:          OrderTypeLimit,
					Price:         "123",
					Quantity:      "50",
					Symbol:        "BTCUSD",
					PostOnly:      false,
					Side:          OrderSideSell,
					ClientOrderID: ClientOrderIdGenerateFast(42),
					Created:       genTimestamp(),
					OrderID:       130173734978,
					OrderStatus:   OrderStatusNew,
				},
				ReportType: ReportTypeNew,
				Timestamp:  genTimestamp(),
			}}
			time.Sleep(time.Millisecond * 100)
			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: OrderModel{
					TimeInForce:   OrderTimeInForceGTC,
					Type:          OrderTypeLimit,
					Price:         "123",
					Quantity:      "50",
					Symbol:        "BTCUSD",
					PostOnly:      false,
					Side:          OrderSideSell,
					ClientOrderID: ClientOrderIdGenerateFast(42),
					Created:       genTimestamp(),
					OrderID:       130173734978,
					OrderStatus:   OrderStatusFilled,
					LastQuantity:  50,
					LastPrice:     "123",
				},
				ReportType: ReportTypeTrade,
				Timestamp:  genTimestamp(),
				TradeID:    1123,
				FeeAmount:  "0.01",
			}}

			return nil
		})

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			order, _, err := trader.NewOrder(ctx, userID, &RequestOrderNew{
				ClientOrderID: clientID,
				TimeInForce:   OrderTimeInForceGTC,
				Symbol:        "BTCUSD",
				Price:         "1",
				Quantity:      "1",
			})
			assert.NilError(t, err)
			assert.Equal(t, clientID, order.ClientOrderID)
			assert.Equal(t, OrderStatusNew, order.OrderStatus)
		}()
		time.Sleep(time.Millisecond * 10)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		report, err := trader.CancelOrder(ctx, userID, clientID)
		assert.NilError(t, err)
		assert.Equal(t, OrderStatusFilled, report.OrderStatus)

		wg.Wait()

	})

	assert.Check(t, push.isMocksEmpty(), "push mock full filled")
	assert.Check(t, xsub.isMocksEmpty(), "xsub mock full filled")
	assert.Check(t, len(trader.pendingFilled) == 0, "new calls is empty")
	assert.Check(t, len(trader.pendingCancel) == 0, "cancel calls is empty")
	assert.Check(t, len(trader.pendingSnap) == 0, "snapshot calls is empty")
}

func TestZmqTrader_ReplaceOrder(t *testing.T) {
	xsub, push, trader := createSweetPair(true)
	<-trader.Ready()
	assert.Check(t, trader.IsReady(), "ready state")
	cleanMocks := func() {
		push.cleanMocks()
		xsub.cleanMocks()
	}
	t.Run("reject order not found in container", func(t *testing.T) {

		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(82)
		cancelID := ClientOrderIdGenerateFast(92)
		xsub.addSubscribeOrders(userID, make([]OrderModel, 0))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := trader.ReplaceOrder(ctx, userID, RequestOrderReplace{
			ClientOrderID:        clientID,
			RequestClientOrderID: cancelID,
			Quantity:             "1",
		})
		assert.Equal(t, ErrorNotFoundOrder, err)
	})

	t.Run("reject order not found in core", func(t *testing.T) {

		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(82)
		cancelID := ClientOrderIdGenerateFast(92)
		orderSnap := OrderModel{
			TimeInForce:   OrderTimeInForceGTC,
			Type:          OrderTypeLimit,
			Price:         "123",
			Quantity:      "50",
			Symbol:        "BTCUSD",
			PostOnly:      false,
			Side:          OrderSideSell,
			ClientOrderID: clientID,
			Created:       genTimestamp(),
			OrderID:       130173734978,
			OrderStatus:   OrderStatusNew,
		}
		xsub.addSubscribeOrders(userID, []OrderModel{orderSnap})

		push.addReplaceMock(func(order requestOrderReplacePayload) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			assert.Equal(t, cancelID, order.CancelRequestClientOrderID)

			xsub.reports <- &payloadReject{
				userID:                     userID,
				ClientOrderID:              clientID,
				CancelRequestClientOrderID: order.CancelRequestClientOrderID,
				RejectReason:               orderRejectNotFound,
			}
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := trader.ReplaceOrder(ctx, userID, RequestOrderReplace{
			ClientOrderID:        clientID,
			RequestClientOrderID: cancelID,
			Quantity:             "1",
		})
		assert.Equal(t, err, ErrorNotFoundOrder)
		assert.ErrorContains(t, err, ErrorNotFoundOrder.Error())
	})

	t.Run("reject order not found in core validate price", func(t *testing.T) {

		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(82)
		cancelID := ClientOrderIdGenerateFast(92)
		orderSnap := OrderModel{
			TimeInForce:   OrderTimeInForceGTC,
			Type:          OrderTypeLimit,
			Price:         "123",
			Quantity:      "50",
			Symbol:        "BTCUSD",
			PostOnly:      false,
			Side:          OrderSideSell,
			ClientOrderID: clientID,
			Created:       genTimestamp(),
			OrderID:       130173734978,
			OrderStatus:   OrderStatusNew,
		}
		xsub.addSubscribeOrders(userID, []OrderModel{orderSnap})

		push.addReplaceMock(func(order requestOrderReplacePayload) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			assert.Equal(t, cancelID, order.CancelRequestClientOrderID)
			assert.Equal(t, "123", order.Price)
			assert.Equal(t, "1", order.Quantity)

			xsub.reports <- &payloadReject{
				userID:                     userID,
				ClientOrderID:              clientID,
				CancelRequestClientOrderID: order.CancelRequestClientOrderID,
				RejectReason:               orderRejectNotFound,
			}
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := trader.ReplaceOrder(ctx, userID, RequestOrderReplace{
			ClientOrderID:        clientID,
			RequestClientOrderID: cancelID,
			Quantity:             "1",
		})
		assert.ErrorContains(t, err, ErrorNotFoundOrder.Error())
	})

	t.Run("reject order not found in core validate quantity", func(t *testing.T) {

		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(82)
		cancelID := ClientOrderIdGenerateFast(92)
		orderSnap := OrderModel{
			TimeInForce:   OrderTimeInForceGTC,
			Type:          OrderTypeLimit,
			Price:         "123",
			Quantity:      "50",
			Symbol:        "BTCUSD",
			PostOnly:      false,
			Side:          OrderSideSell,
			ClientOrderID: clientID,
			Created:       genTimestamp(),
			OrderID:       130173734978,
			OrderStatus:   OrderStatusNew,
		}
		xsub.addSubscribeOrders(userID, []OrderModel{orderSnap})

		push.addReplaceMock(func(order requestOrderReplacePayload) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			assert.Equal(t, cancelID, order.CancelRequestClientOrderID)
			assert.Equal(t, "1", order.Price)
			assert.Equal(t, "50", order.Quantity)

			xsub.reports <- &payloadReject{
				userID:                     userID,
				ClientOrderID:              clientID,
				CancelRequestClientOrderID: order.CancelRequestClientOrderID,
				RejectReason:               orderRejectNotFound,
			}
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := trader.ReplaceOrder(ctx, userID, RequestOrderReplace{
			ClientOrderID:        clientID,
			RequestClientOrderID: cancelID,
			Price:                "1",
		})
		assert.ErrorContains(t, err, ErrorNotFoundOrder.Error())
	})

	t.Run("reject order not changed", func(t *testing.T) {

		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(82)
		cancelID := ClientOrderIdGenerateFast(92)
		orderSnap := OrderModel{
			TimeInForce:   OrderTimeInForceGTC,
			Type:          OrderTypeLimit,
			Price:         "123",
			Quantity:      "50",
			Symbol:        "BTCUSD",
			PostOnly:      false,
			Side:          OrderSideSell,
			ClientOrderID: clientID,
			Created:       genTimestamp(),
			OrderID:       130173734978,
			OrderStatus:   OrderStatusNew,
		}
		xsub.addSubscribeOrders(userID, []OrderModel{orderSnap})

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := trader.ReplaceOrder(ctx, userID, RequestOrderReplace{
			ClientOrderID:        clientID,
			RequestClientOrderID: cancelID,
		})
		assert.ErrorContains(t, err, ErrorNotChanged.Error())
	})

	t.Run("success replaced full flow", func(t *testing.T) {

		cleanMocks()
		userID := genUserID()
		clientID := ClientOrderIdGenerateFast(82)
		cancelID := ClientOrderIdGenerateFast(92)
		orderSnap := OrderModel{
			TimeInForce:   OrderTimeInForceGTC,
			Type:          OrderTypeLimit,
			Price:         "123",
			Quantity:      "50",
			Symbol:        "BTCUSD",
			PostOnly:      false,
			Side:          OrderSideSell,
			ClientOrderID: clientID,
			Created:       genTimestamp(),
			OrderID:       130173734978,
			OrderStatus:   OrderStatusNew,
		}
		xsub.addSubscribeOrders(userID, []OrderModel{orderSnap})

		push.addReplaceMock(func(order requestOrderReplacePayload) error {
			assert.Equal(t, messageUserID(userID), order.UserID)
			assert.Equal(t, clientID, order.ClientOrderID)
			assert.Equal(t, cancelID, order.CancelRequestClientOrderID)
			assert.Equal(t, "124", order.Price)
			assert.Equal(t, "50", order.Quantity)

			xsub.reports <- &payloadReport{userID: userID, report: OrderReport{
				OrderModel: OrderModel{
					TimeInForce:   OrderTimeInForceGTC,
					Type:          OrderTypeLimit,
					Price:         "124",
					Quantity:      "50",
					Symbol:        "BTCUSD",
					PostOnly:      false,
					Side:          OrderSideSell,
					ClientOrderID: cancelID,
					Created:       genTimestamp(),
					OrderID:       130173734975,
					LatestOrderID: 130173734979,
					OrderStatus:   OrderStatusNew,
					LastQuantity:  50,
					LastPrice:     "123",
				},
				OriginalClientOrderID: clientID,
				ReportType:            ReportTypeReplaced,
				Timestamp:             genTimestamp(),
			}}
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		res, err := trader.ReplaceOrder(ctx, userID, RequestOrderReplace{
			ClientOrderID:        clientID,
			RequestClientOrderID: cancelID,
			Price:                "124",
		})
		assert.NilError(t, err)
		assert.Equal(t, OrderStatusNew, res.OrderStatus)
		assert.Equal(t, cancelID, res.ClientOrderID)
		assert.Equal(t, uint64(130173734979), res.LatestOrderID)

		newOrder, err := trader.GetOrder(ctx, userID, cancelID)
		assert.NilError(t, err)
		assert.Equal(t, uint64(130173734975), newOrder.OrderID)
		assert.Equal(t, uint64(130173734979), newOrder.LatestOrderID)

		_, err = trader.GetOrder(ctx, userID, clientID)
		assert.ErrorContains(t, err, ErrorNotFoundOrder.Error())

	})

}
