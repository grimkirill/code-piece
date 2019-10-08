package trading

import (
	"context"
	"sync"
	"time"

	"sync/atomic"

	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var readyState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "zmq_ready_state",
	Help: "Zmq trading gate status",
}, []string{"gate"})

var requestDurations = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Name:       "zmq_request_duration_us",
	Help:       "zmq request durations microseconds",
	AgeBuckets: 1,
}, []string{"gate", "action"})

func init() {
	prometheus.MustRegister(readyState, requestDurations)
}

type identifier [40]byte

func getIdentifier(userID uint64, clientOrderID ClientOrderId) (id identifier) {
	binary.LittleEndian.PutUint64(id[:], userID)
	copy(id[8:], clientOrderID[:])
	return
}

// zmqTrader trading with one zmq_transactional gate connected
type zmqTrader struct {
	logger        *zap.Logger
	push          pushConnecter
	xsub          xsubConnecter
	mx            sync.Mutex
	pendingFilled map[identifier]*requestCall
	pendingSnap   map[uint64]map[uint64]*callSnapshot
	pendingCancel map[identifier]map[uint64]*requestCall
	isReady       uint32
	isCanaryTrade bool
	canaryTrade   chan bool
	ready         chan bool
	acceptedUsers sync.Map
	orders        *ordersContainer
}

// Unsubscribe flush orders cache and unsubscribe zmq
func (c *zmqTrader) Unsubscribe(userID uint64) {
	err := c.xsub.UnSubscribe(userID)
	if err != nil {
		c.logger.Error("fail send unsubscribe", zap.Error(err))
	}
	c.acceptedUsers.Delete(userID)
	c.orders.delete(userID)
}

func (c *zmqTrader) ensureUserAccepted(userID uint64) {
	c.acceptedUsers.LoadOrStore(userID, true)
}

func (c *zmqTrader) requireSnapshot(ctx context.Context, userID uint64) error {
	c.ensureUserAccepted(userID)

	if c.orders.hasOrders(userID) {
		return nil
	}

	call := createCallSnapshot(c.xsub.GetAddr())
	var duplicateRequest bool
	c.mx.Lock()
	if _, duplicateRequest = c.pendingSnap[userID]; !duplicateRequest {
		c.pendingSnap[userID] = make(map[uint64]*callSnapshot)
	}
	c.pendingSnap[userID][call.id] = call
	c.mx.Unlock()

	defer func() {
		c.mx.Lock()
		delete(c.pendingSnap[userID], call.id)
		if len(c.pendingSnap[userID]) == 0 {
			delete(c.pendingSnap, userID)
		}
		c.mx.Unlock()
	}()

	if !duplicateRequest {
		err := c.xsub.Subscribe(userID)
		if err != nil {
			return err
		}
	}

	select {
	case result := <-call.Done:
		return result.Error
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *zmqTrader) requireSnapshotOrder(ctx context.Context, userID uint64, clientOrderID ClientOrderId) (*OrderModel, error) {
	if !c.orders.hasOrders(userID) {
		err := c.requireSnapshot(ctx, userID)
		if err != nil {
			return nil, err
		}
	}

	order, ok := c.orders.getOrder(userID, clientOrderID)
	if ok {
		return &order, nil
	}

	return nil, nil
}

// NewOrder place new order for user
func (c *zmqTrader) NewOrder(ctx context.Context, userID uint64, order *RequestOrderNew) (*OrderReport, []Trade, error) {
	existOrder, err := c.requireSnapshotOrder(ctx, userID, order.ClientOrderID)
	if err != nil {
		return nil, nil, err
	}
	if existOrder != nil {
		return nil, nil, ErrorDuplicateClientOrder
	}

	call := createCall(c.xsub.GetAddr())
	call.waitComplete = order.WaitComplete
	call.symbol = order.Symbol
	call.side = order.Side

	var duplicate bool
	id := getIdentifier(userID, order.ClientOrderID)

	c.mx.Lock()
	if _, duplicate = c.pendingFilled[id]; !duplicate {
		c.pendingFilled[id] = call
	}
	c.mx.Unlock()

	if duplicate {
		return nil, nil, ErrorDuplicateClientOrder
	}
	defer func() {
		c.mx.Lock()
		delete(c.pendingFilled, id)
		c.mx.Unlock()
	}()

	//send request for socket
	err = c.push.SendRequestNew(order.getPayload(userID))
	if err != nil {
		return nil, nil, err
	}

	select {
	case result := <-call.Done:
		return result.Reply, result.trades, result.Error
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}

// CancelOrder cancel order by its trader order id for user
func (c *zmqTrader) CancelOrder(ctx context.Context, userID uint64, clientOrderID ClientOrderId) (*OrderReport, error) {
	existOrder, err := c.requireSnapshotOrder(ctx, userID, clientOrderID)
	if err != nil {
		return nil, err
	}
	var requestSymbol string
	var requestSide OrderSide
	id := getIdentifier(userID, clientOrderID)

	if existOrder == nil {
		c.mx.Lock()
		pending, ok := c.pendingFilled[id]
		c.mx.Unlock()
		if !ok {
			return nil, ErrorNotFoundOrder
		}
		requestSymbol = pending.symbol
		requestSide = pending.side
	} else {
		requestSymbol = existOrder.Symbol
		requestSide = existOrder.Side
	}

	cancelRequest := requestOrderCancel{
		UserID:                     messageUserID(userID),
		ClientOrderID:              clientOrderID,
		CancelRequestClientOrderID: ClientOrderIdGenerate(),
		Symbol:                     requestSymbol,
		Side:                       requestSide,
	}

	call := createCall(c.xsub.GetAddr())
	call.CancelRequestClientOrderID = cancelRequest.CancelRequestClientOrderID
	c.mx.Lock()
	if _, ok := c.pendingCancel[id]; !ok {
		c.pendingCancel[id] = make(map[uint64]*requestCall)
	}
	c.pendingCancel[id][call.id] = call
	c.mx.Unlock()
	defer func() {
		c.mx.Lock()
		delete(c.pendingCancel[id], call.id)
		if len(c.pendingCancel[id]) == 0 {
			delete(c.pendingCancel, id)
		}
		c.mx.Unlock()
	}()

	//send request for socket
	err = c.push.SendRequestCancel(cancelRequest)
	if err != nil {
		return nil, err
	}

	select {
	case result := <-call.Done:
		return result.Reply, result.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ReplaceOrder replace user order with new params
func (c *zmqTrader) ReplaceOrder(ctx context.Context, userID uint64, req RequestOrderReplace) (*OrderReport, error) {
	existOrder, err := c.requireSnapshotOrder(ctx, userID, req.ClientOrderID)
	if err != nil {
		return nil, err
	}
	if existOrder == nil {
		return nil, ErrorNotFoundOrder
	}
	id := getIdentifier(userID, req.ClientOrderID)

	requestOrderReplace := requestOrderReplacePayload{
		UserID:                     messageUserID(userID),
		ClientOrderID:              req.ClientOrderID,
		CancelRequestClientOrderID: req.RequestClientOrderID,
		Symbol:                     existOrder.Symbol,
		Type:                       existOrder.Type,
		TimeInForce:                existOrder.TimeInForce,
		Side:                       existOrder.Side,
	}

	if req.Price != "" {
		requestOrderReplace.Price = req.Price
	} else {
		requestOrderReplace.Price = existOrder.Price
	}

	if req.Quantity != "" {
		requestOrderReplace.Quantity = req.Quantity
	} else {
		requestOrderReplace.Quantity = existOrder.Quantity
	}

	if requestOrderReplace.Quantity == existOrder.Quantity && requestOrderReplace.Price == existOrder.Price {
		return nil, ErrorNotChanged
	}

	call := createCall(c.xsub.GetAddr())
	call.CancelRequestClientOrderID = requestOrderReplace.CancelRequestClientOrderID

	c.mx.Lock()
	if _, ok := c.pendingCancel[id]; !ok {
		c.pendingCancel[id] = make(map[uint64]*requestCall)
	}
	c.pendingCancel[id][call.id] = call
	c.mx.Unlock()

	defer func() {
		c.mx.Lock()
		delete(c.pendingCancel[id], call.id)
		if len(c.pendingCancel[id]) == 0 {
			delete(c.pendingCancel, id)
		}
		c.mx.Unlock()
	}()

	//send request for socket
	err = c.push.SendRequestReplace(requestOrderReplace)
	if err != nil {
		return nil, err
	}

	select {
	case result := <-call.Done:
		return result.Reply, result.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// GetSnapshot get user orders snapshot
func (c *zmqTrader) GetSnapshot(ctx context.Context, userID uint64) ([]OrderModel, error) {
	err := c.requireSnapshot(ctx, userID)
	if err != nil {
		return nil, err
	}

	ordersList, ok := c.orders.getOrders(userID)
	if ok {
		return ordersList, nil
	}

	return nil, errors.New("fail processing consist user snapshot request")
}

func (c *zmqTrader) GetOrder(ctx context.Context, userID uint64, clientOrderID ClientOrderId) (*OrderModel, error) {
	existOrder, err := c.requireSnapshotOrder(ctx, userID, clientOrderID)
	if err != nil {
		return nil, err
	}
	if existOrder == nil {
		return nil, ErrorNotFoundOrder
	}
	return existOrder, nil
}

func (c *zmqTrader) input() {
	for report := range c.xsub.Reports() {
		if orderReport, ok := report.(*payloadReport); ok {
			c.handleReport(orderReport)
		} else if reject, ok := report.(*payloadReject); ok {
			c.handleRejectPayload(reject)
		} else if snap, ok := report.(*payloadSnapshot); ok {
			c.handleSnapshot(snap)
		} else {
			c.logger.Error("unexpected input report", zap.Reflect("report", report))
			panic("unexpected input report")
		}
	}
}

func (c *zmqTrader) handleRejectPayload(reject *payloadReject) {
	c.mx.Lock()
	defer c.mx.Unlock()
	id := getIdentifier(reject.userID, reject.ClientOrderID)
	pending := c.pendingFilled[id]
	if pending != nil {
		pending.Error = getErrorByReject(reject.RejectReason)
		if pending.Error == ErrorUnknown {
			c.logger.Error("zmq: unexpected error reject", zap.String("reason", reject.RejectReason))
		}
		pending.done()
	}

	if calls, ok := c.pendingCancel[id]; ok {
		for _, item := range calls {
			if item.CancelRequestClientOrderID == reject.CancelRequestClientOrderID {
				item.Error = getErrorByReject(reject.RejectReason)
				if item.Error == ErrorUnknown {
					c.logger.Error("zmq: unexpected error reject", zap.String("reason", reject.RejectReason))
				}
				item.done()
			}
		}
	}
}

func (c *zmqTrader) handleSnapshot(snap *payloadSnapshot) {
	c.logger.Info("zmq trader: handle snapshot", zap.Uint64("user", snap.userID), zap.Int("orders", len(snap.orders)))
	_, ok := c.acceptedUsers.Load(snap.userID)
	if !ok {
		return
	}

	c.orders.setOrders(snap.userID, snap.orders)

	c.mx.Lock()
	defer c.mx.Unlock()
	if calls, ok := c.pendingSnap[snap.userID]; ok {
		for _, call := range calls {
			call.done()
		}
	}
}

func (c *zmqTrader) handleReportFillSnapshot(userID uint64, orderReport OrderReport) {

	if orderReport.ReportType == ReportTypeNew ||
		orderReport.ReportType == ReportTypeSuspended ||
		orderReport.ReportType == ReportTypeCanceled ||
		orderReport.ReportType == ReportTypeExpired ||
		orderReport.ReportType == ReportTypeTrade ||
		orderReport.ReportType == ReportTypeReplaced {
		c.orders.handleReport(userID, orderReport)
	}
}

func (c *zmqTrader) handleReportReplyCall(userID uint64, orderReport OrderReport) {
	id := getIdentifier(userID, orderReport.ClientOrderID)
	c.mx.Lock()
	defer c.mx.Unlock()
	if orderReport.ReportType == ReportTypeCanceled {

		if calls, ok := c.pendingCancel[id]; ok {
			for _, call := range calls {
				call.Reply = &orderReport
				call.done()
			}
		}

		return
	}
	if orderReport.ReportType == ReportTypeNew ||
		orderReport.ReportType == ReportTypeSuspended ||
		orderReport.ReportType == ReportTypeExpired ||
		orderReport.ReportType == ReportTypeTrade ||
		orderReport.ReportType == ReportTypeReplaced {

		pending := c.pendingFilled[id]
		if pending != nil {
			pending.Reply = &orderReport

			if orderReport.ReportType == ReportTypeTrade {
				pending.trades = append(pending.trades, Trade{
					ID:        orderReport.TradeID,
					Timestamp: orderReport.Timestamp,
					Quantity:  orderReport.LastQuantity,
					Price:     orderReport.LastPrice,
					FeeAmount: orderReport.FeeAmount,
				})
			}

			if orderReport.TimeInForce == OrderTimeInForceFOK || orderReport.TimeInForce == OrderTimeInForceIOC {
				//skip resolving incomplete orders
				if !pending.waitComplete || orderReport.OrderStatus == OrderStatusFilled || orderReport.OrderStatus == OrderStatusExpired {
					pending.done()
				}
			} else {
				c.logger.Info("report", zap.String("type", orderReport.ReportType.String()), zap.String("clientOrderId", orderReport.ClientOrderID.String()))
				pending.done()
			}
		}

		// resolve cancel request by filled orders
		// todo integration tests for replace request
		if orderReport.OrderStatus == OrderStatusFilled || orderReport.ReportType == ReportTypeReplaced {

			if orderReport.ReportType == ReportTypeReplaced {
				if calls, ok := c.pendingCancel[getIdentifier(userID, orderReport.OriginalClientOrderID)]; ok {
					for _, call := range calls {
						call.Reply = &orderReport
						call.done()
					}
				}
			} else {
				if calls, ok := c.pendingCancel[id]; ok {
					for _, call := range calls {
						call.Reply = &orderReport
						call.done()
					}
				}
			}

		}
	}
}

func (c *zmqTrader) handleReport(payload *payloadReport) {
	_, ok := c.acceptedUsers.Load(payload.userID)
	if ok {
		c.handleReportFillSnapshot(payload.userID, payload.report)
	}
	c.handleReportReplyCall(payload.userID, payload.report)
}

func (c *zmqTrader) setReady(val bool) {
	var promStatus float64
	var state uint32
	if val {
		promStatus = 1
		state = 1
	}
	readyState.WithLabelValues(c.xsub.GetAddr()).Set(promStatus)

	if atomic.SwapUint32(&c.isReady, state) != state {
		select {
		case c.ready <- val:
			// ok
		default:
			// We don't want to block here. It is the caller's responsibility to make
			// sure the channel has enough buffer space. See comment in Go().
			c.logger.Error("zmq ready call discarding due to insufficient chan capacity")
		}
	}
}

func (c *zmqTrader) IsReady() bool {
	return atomic.LoadUint32(&c.isReady) == 1
}

func (c *zmqTrader) Ready() chan bool {
	return c.ready
}

func (c *zmqTrader) canaryTestCall() bool {
	ctx, cancel := context.WithTimeout(context.Background(), canaryTestDuration)
	defer cancel()
	requestOrder := RequestOrderNew{
		ClientOrderID: ClientOrderIdGenerate(),
		Symbol:        canaryTestSymbol,
		Price:         canaryTestPrice,
		Quantity:      canaryTestQuantity,
		Side:          OrderSideSell,
		Type:          OrderTypeLimit,
		TimeInForce:   OrderTimeInForceFOK,
	}
	res, _, err := c.NewOrder(ctx, canaryUser, &requestOrder)
	if err != nil {
		if reject, ok := err.(ErrorResponse); ok {
			if reject == ErrorExceedsLimit || reject == ErrorTradingNotStarted || reject == ErrorTooLateToEnter {
				c.logger.Warn("zmq trading: canary ping with warning", zap.Error(err), zap.String("gate", c.xsub.GetAddr()))
				return true
			}
		}
		c.logger.Warn("zmq trading: canary ping fail", zap.Error(err), zap.String("gate", c.xsub.GetAddr()))
		return false
	}
	if res.OrderStatus == OrderStatusNew || res.OrderStatus == OrderStatusExpired {
		return true
	}

	c.logger.Warn("zmq trading: canary ping fail", zap.String("order status", res.OrderStatus.String()), zap.String("gate", c.xsub.GetAddr()))
	return false
}

func (c *zmqTrader) canaryTest() {
	result := c.canaryTestCall()
	if result != c.isCanaryTrade {
		c.isCanaryTrade = result
		c.canaryTrade <- result
	}
}

func (c *zmqTrader) canaryTestLoop() {
	for range time.NewTicker(canaryTestDuration).C {
		if c.push.IsReady() && c.xsub.IsReady() {
			c.canaryTest()
		}
	}
}

func (c *zmqTrader) handleReady() {
	for {
		select {
		case pushReady := <-c.push.Ready():
			c.logger.Info("zmq trading:", zap.Bool("push ready state", pushReady), zap.String("gate", c.xsub.GetAddr()))
		case xsubReady := <-c.xsub.Ready():
			c.logger.Info("zmq trading:", zap.Bool("xsub ready state", xsubReady), zap.String("gate", c.xsub.GetAddr()))

			if xsubReady {
				go c.canaryTest()
			}
		case canaryResult := <-c.canaryTrade:
			c.logger.Info("zmq trading:", zap.Bool("canary trade result", canaryResult), zap.String("gate", c.xsub.GetAddr()))
		}

		isReady := c.push.IsReady() && c.xsub.IsReady() && c.isCanaryTrade
		c.setReady(isReady)
	}
}

// StartNewClient create trading trader and init it
func StartNewClient(logger *zap.Logger, push pushConnecter, xsub xsubConnecter) *zmqTrader {
	client := &zmqTrader{
		logger:        logger,
		push:          push,
		xsub:          xsub,
		pendingFilled: make(map[identifier]*requestCall),
		pendingCancel: make(map[identifier]map[uint64]*requestCall),
		pendingSnap:   make(map[uint64]map[uint64]*callSnapshot),
		ready:         make(chan bool, 2),
		canaryTrade:   make(chan bool),
		orders:        newOrderContainer(),
	}
	go client.input()
	go client.handleReady()
	go client.canaryTestLoop()
	return client
}
