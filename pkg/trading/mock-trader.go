package trading

import (
	"context"

	"sync/atomic"

	"sync"

	"time"

	"strconv"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type expectation struct {
	reqId       ClientOrderId
	report      *OrderReport
	err         error
	Unsubscribe bool
	GetSnapshot bool
	GetOrder    bool
}

type MockTrader struct {
	logger         *zap.Logger
	isReady        uint32
	ready          chan bool
	orders         *ordersContainer
	allowAutoTrade sync.Map
	expectationMx  sync.Mutex
	expectations   map[uint64][]expectation
}

func (m *MockTrader) addExpectation(userID uint64, expect expectation) {
	m.expectationMx.Lock()
	defer m.expectationMx.Unlock()
	list, ok := m.expectations[userID]
	if !ok {
		m.expectations[userID] = []expectation{expect}
	} else {
		m.expectations[userID] = append(list, expect)
	}
}

func (m *MockTrader) isEmptyExpectations() bool {
	m.expectationMx.Lock()
	defer m.expectationMx.Unlock()
	return len(m.expectations) == 0
}

func (m *MockTrader) hasExpectations(userID uint64) (ok bool) {
	m.expectationMx.Lock()
	defer m.expectationMx.Unlock()
	_, ok = m.expectations[userID]
	return
}

func (m *MockTrader) popExpectations(userID uint64) (expectation, error) {
	m.expectationMx.Lock()
	defer m.expectationMx.Unlock()
	list, ok := m.expectations[userID]
	if !ok {
		err := errors.New("expectations not found for user " + strconv.FormatUint(userID, 10))
		m.logger.Error("expectations", zap.Error(err))
		return expectation{}, err
	}
	if len(list) == 0 {
		err := errors.New("expectations is empty for user " + strconv.FormatUint(userID, 10))
		m.logger.Error("expectations", zap.Error(err))
		return expectation{}, err
	}
	m.expectations[userID] = list[1:]
	if len(m.expectations[userID]) == 0 {
		delete(m.expectations, userID)
	}
	return list[0], nil
}

func (m *MockTrader) Unsubscribe(userID uint64) {
	m.orders.delete(userID)
	m.logger.Info("mock-trader:", zap.Uint64("unsubscribe", userID))
	if m.hasExpectations(userID) {
		exp, _ := m.popExpectations(userID)
		if !exp.Unsubscribe {
			m.logger.Warn("unexpected mock for unsubscribe", zap.Reflect("mock", exp))
			return
		}
	}
}

func (m *MockTrader) GetSnapshot(ctx context.Context, userID uint64) ([]OrderModel, error) {
	if m.hasExpectations(userID) {
		exp, err := m.popExpectations(userID)
		if err != nil {
			return nil, err
		}
		if !exp.GetSnapshot {
			m.logger.Warn("unexpected mock for got snapshot", zap.Reflect("mock", exp))
			return nil, errors.New("unexpected mock")
		}
		if !m.orders.hasOrders(userID) {
			m.orders.setOrders(userID, nil)
		}
		if exp.report != nil {
			m.orders.handleReport(userID, *exp.report)
		}
	}

	if !m.orders.hasOrders(userID) {
		return nil, errors.New("user not inited")
	}
	orders, _ := m.orders.getOrders(userID)
	return orders, nil
}

func (m *MockTrader) GetOrder(ctx context.Context, userID uint64, clientOrderID ClientOrderId) (*OrderModel, error) {
	if !m.orders.hasOrders(userID) {
		return nil, errors.New("user not inited")
	}
	order, ok := m.orders.getOrder(userID, clientOrderID)
	if !ok {
		return nil, ErrorNotFoundOrder
	}
	return &order, nil
}

func (m *MockTrader) NewOrder(ctx context.Context, userID uint64, order *RequestOrderNew) (*OrderReport, []Trade, error) {
	if m.hasExpectations(userID) {
		exp, err := m.popExpectations(userID)
		if err != nil {
			return nil, nil, err
		}
		if exp.reqId != order.ClientOrderID {
			m.logger.Warn("unexpected mock for new order", zap.Reflect("mock", exp), zap.Reflect("request", order))
			return nil, nil, errors.New("unexpected mock")
		}
		if !m.orders.hasOrders(userID) {
			m.orders.setOrders(userID, nil)
		}
		if exp.report != nil {
			m.orders.handleReport(userID, *exp.report)
		}

		return exp.report, nil, exp.err
	}

	if !m.orders.hasOrders(userID) {
		return nil, nil, errors.New("user not inited")
	}
	if _, ok := m.allowAutoTrade.Load(userID); ok {
		m.logger.Info("mock-trader: auto trade new order", zap.Uint64("user", userID), zap.Reflect("request", order))
		if _, hasOrder := m.orders.getOrder(userID, order.ClientOrderID); hasOrder {
			return nil, nil, ErrorDuplicateClientOrder
		}
		report := m.processAutoOrderReport(userID, order)
		return &report, nil, nil
	}
	return nil, nil, errors.New("not implemented")
}

func (m *MockTrader) CancelOrder(ctx context.Context, userID uint64, clientOrderID ClientOrderId) (*OrderReport, error) {
	if !m.orders.hasOrders(userID) {
		return nil, errors.New("user not inited")
	}
	if _, ok := m.allowAutoTrade.Load(userID); ok {
		m.logger.Info("mock-trader: auto trade cancel order", zap.Uint64("user", userID), zap.String("clientOrderId", clientOrderID.String()))
		order, hasOrder := m.orders.getOrder(userID, clientOrderID)
		if !hasOrder {
			return nil, ErrorNotFoundOrder
		}

		report := OrderReport{
			OrderModel: order,
			ReportType: ReportTypeCanceled,
			Timestamp:  uint64(time.Now().UnixNano() / 1e6),
		}
		report.OrderStatus = OrderStatusCanceled
		m.orders.handleReport(userID, report)
		return &report, nil
	}

	return nil, errors.New("not implemented")
}

func (m *MockTrader) ReplaceOrder(ctx context.Context, userID uint64, req RequestOrderReplace) (*OrderReport, error) {
	return nil, errors.New("not implemented")
}

func (m *MockTrader) IsReady() bool {
	return atomic.LoadUint32(&m.isReady) == 1
}

func (m *MockTrader) Ready() chan bool {
	return m.ready
}

func (m *MockTrader) SetReady(val bool) {
	setVal := uint32(0)
	if val {
		setVal = 1
	}

	if atomic.SwapUint32(&m.isReady, setVal) != setVal {
		m.logger.Info("mock-trader:", zap.Bool("set ready", val))
		m.ready <- val
	}
}

func NewMockTrader(logger *zap.Logger) *MockTrader {
	client := &MockTrader{
		logger:       logger,
		ready:        make(chan bool, 2),
		orders:       newOrderContainer(),
		expectations: make(map[uint64][]expectation),
	}
	logger.Info("mock-trader: created")
	return client
}

var nextAutoOrderId uint64

func getNextAutoOrderId() uint64 {
	return atomic.AddUint64(&nextAutoOrderId, 1)
}

func (m *MockTrader) processAutoOrderReport(userID uint64, req *RequestOrderNew) OrderReport {
	report := OrderReport{
		OrderModel: OrderModel{
			ClientOrderID: req.ClientOrderID,
			Type:          req.Type,
			Side:          req.Side,
			OrderStatus:   OrderStatusNew,
			TimeInForce:   req.TimeInForce,
			Price:         req.Price,
			Symbol:        req.Symbol,
			Quantity:      req.Quantity,
			PostOnly:      req.PostOnly,
			LatestOrderID: getNextAutoOrderId(),
			OrderID:       getNextAutoOrderId(),
			Created:       uint64(time.Now().UnixNano() / 1e6),
			CumQuantity:   "0",
		},
		ReportType: ReportTypeNew,
		Timestamp:  uint64(time.Now().UnixNano() / 1e6),
	}
	m.orders.handleReport(userID, report)

	if req.TimeInForce == OrderTimeInForceIOC || req.TimeInForce == OrderTimeInForceFOK {
		expired := report
		expired.OrderStatus = OrderStatusExpired
		expired.ReportType = ReportTypeExpired
		m.orders.handleReport(userID, expired)
	}

	return report
}

func (m *MockTrader) SetupFixtures() {
	k := 0
	created := uint64(1564755682000)
	orders := make([]OrderModel, 0)
	for _, oType := range []OrderType{OrderTypeMarket, OrderTypeLimit, OrderTypeStopMarket, OrderTypeStopLimit} {
		for _, oSide := range []OrderSide{OrderSideSell, OrderSideBuy} {
			k++
			order := OrderModel{
				ClientOrderID: ClientOrderIdGenerateFast(k),
				Type:          oType,
				Side:          oSide,
				OrderStatus:   OrderStatusNew,
				TimeInForce:   OrderTimeInForceGTC,
				Price:         "10525.20",
				Symbol:        "BTCUSD",
				Quantity:      "100",
				PostOnly:      false,
				LatestOrderID: getNextAutoOrderId(),
				OrderID:       getNextAutoOrderId(),
				Created:       created + uint64(k),
				CumQuantity:   "0",
			}
			orders = append(orders, order)
		}
	}
	userID := uint64(10101)
	m.logger.Info("mock-trader: setup fixtures", zap.Uint64("user", userID))
	m.orders.setOrders(userID, orders)
	m.allowAutoTrade.Store(userID, true)

	userEmpty := uint64(10102)
	m.logger.Info("mock-trader: setup empty user", zap.Uint64("user", userEmpty))
	m.orders.setOrders(userEmpty, make([]OrderModel, 0))
	m.allowAutoTrade.Store(userEmpty, true)
}

func (m *MockTrader) StartHttpServer(addr string) {

}
