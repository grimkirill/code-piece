package trading

import (
	"sync"
)

type ordersContainer struct {
	locks sync.Map
	users map[uint64]map[ClientOrderId]OrderModel
}

func newOrderContainer() *ordersContainer {
	return &ordersContainer{
		users: make(map[uint64]map[ClientOrderId]OrderModel),
	}
}

func (con *ordersContainer) setOrders(userID uint64, orders []OrderModel) {
	mxVal, ok := con.locks.Load(userID)
	if !ok {
		mxVal, _ = con.locks.LoadOrStore(userID, &sync.RWMutex{})
	}
	mx, _ := mxVal.(*sync.RWMutex)
	mx.Lock()
	defer mx.Unlock()
	userOrders := make(map[ClientOrderId]OrderModel, len(orders))
	for _, order := range orders {
		userOrders[order.ClientOrderID] = order
	}
	con.users[userID] = userOrders
}

func (con *ordersContainer) hasOrders(userID uint64) bool {
	_, ok := con.locks.Load(userID)
	return ok
}

func (con *ordersContainer) getOrders(userID uint64) ([]OrderModel, bool) {
	mxVal, ok := con.locks.Load(userID)
	if !ok {
		return nil, false
	}
	mx, _ := mxVal.(*sync.RWMutex)
	mx.RLock()
	defer mx.RUnlock()

	ordersData := con.users[userID]
	result := make([]OrderModel, 0, len(ordersData))
	for _, order := range ordersData {
		result = append(result, order)
	}
	return result, true
}

// getOrder get one order by its clientOrderId
func (con *ordersContainer) getOrder(userID uint64, id ClientOrderId) (OrderModel, bool) {
	mxVal, ok := con.locks.Load(userID)
	if !ok {
		return OrderModel{}, false
	}
	mx, _ := mxVal.(*sync.RWMutex)
	mx.RLock()
	defer mx.RUnlock()

	orderData, ok := con.users[userID][id]
	return orderData, ok
}

// handleReport fill user orders map. allow process only if user orders snapshot exist
func (con *ordersContainer) handleReport(userID uint64, report OrderReport) {
	if report.ReportType != ReportTypeNew &&
		report.ReportType != ReportTypeSuspended &&
		report.ReportType != ReportTypeCanceled &&
		report.ReportType != ReportTypeExpired &&
		report.ReportType != ReportTypeTrade &&
		report.ReportType != ReportTypeReplaced {
		return
	}
	mxVal, ok := con.locks.Load(userID)
	if !ok {
		return
	}
	mx, _ := mxVal.(*sync.RWMutex)
	mx.Lock()
	defer mx.Unlock()

	if report.ReportType == ReportTypeReplaced {
		delete(con.users[userID], report.OriginalClientOrderID)
	}
	if report.OrderStatus == OrderStatusNew || report.OrderStatus == OrderStatusPartiallyFilled || report.OrderStatus == OrderStatusSuspended {
		con.users[userID][report.ClientOrderID] = report.OrderModel
	} else {
		delete(con.users[userID], report.ClientOrderID)
	}
}

func (con *ordersContainer) delete(userID uint64) {
	mxVal, ok := con.locks.Load(userID)
	if !ok {
		return
	}
	mx, _ := mxVal.(*sync.RWMutex)
	mx.Lock()
	defer mx.Unlock()

	delete(con.users, userID)
	con.locks.Delete(userID)
}
