package trading

import (
	"testing"

	"time"

	"context"

	"go.uber.org/zap"
	"gotest.tools/assert"
)

func TestClusterTrader_Ready(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mock01 := NewMockTrader(logger)
	mock02 := NewMockTrader(logger)
	mock03 := NewMockTrader(logger)

	cluster := newClusterTrader(logger, []Trader{mock01, mock02, mock03})
	assert.Equal(t, cluster.IsReady(), false)

	mock01.SetReady(true)
	<-cluster.Ready()
	assert.Equal(t, cluster.IsReady(), true)

	mock01.SetReady(false)
	<-cluster.Ready()
	assert.Equal(t, cluster.IsReady(), false)

	//turn on
	mock01.SetReady(true)
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, len(cluster.Ready()), 1)
	<-cluster.Ready()

	assert.Equal(t, cluster.IsReady(), true)
	mock02.SetReady(true)
	assert.Equal(t, cluster.IsReady(), true)
	mock03.SetReady(true)
	assert.Equal(t, cluster.IsReady(), true)

	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, len(cluster.Ready()), 0)

	mock03.SetReady(false)
	assert.Equal(t, cluster.IsReady(), true)
	mock02.SetReady(false)
	assert.Equal(t, cluster.IsReady(), true)
	mock01.SetReady(false)
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, len(cluster.Ready()), 1)
	<-cluster.Ready()
	assert.Equal(t, cluster.IsReady(), false)

	//turn on
	mock01.SetReady(true)
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, len(cluster.Ready()), 1)
	<-cluster.Ready()
	assert.Equal(t, cluster.IsReady(), true)
}

func TestClusterTrader_ReadyOverCapacity(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mock01 := NewMockTrader(logger)
	mock02 := NewMockTrader(logger)

	cluster := newClusterTrader(logger, []Trader{mock01, mock02})
	assert.Equal(t, cluster.IsReady(), false)

	for i := 0; i < 3; i++ {
		mock01.SetReady(true)
		time.Sleep(time.Millisecond)
		mock01.SetReady(false)
		time.Sleep(time.Millisecond)
	}

	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, cap(cluster.Ready()), 2)
	assert.Equal(t, len(cluster.Ready()), 2)
}

func TestClusterTrader_GetSnapshot(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mock01 := NewMockTrader(logger)
	mock02 := NewMockTrader(logger)

	userID := uint64(2001234)

	cluster := newClusterTrader(logger, []Trader{mock01, mock02})
	assert.Equal(t, cluster.IsReady(), false)

	t.Run("fail get gate", func(t *testing.T) {
		_, err := cluster.GetSnapshot(context.Background(), userID)
		assert.ErrorContains(t, err, "available")
	})

	mock01.SetReady(true)
	mock02.SetReady(true)

	time.Sleep(time.Millisecond)

	mock01.addExpectation(userID, expectation{GetSnapshot: true})

	orders, err := cluster.GetSnapshot(context.Background(), userID)
	assert.NilError(t, err)
	assert.Equal(t, len(orders), 0)
	assert.Check(t, mock01.isEmptyExpectations())
	assert.Check(t, mock02.isEmptyExpectations())

	mock01.addExpectation(userID, expectation{Unsubscribe: true})
	mock02.addExpectation(userID, expectation{GetSnapshot: true})

	mock01.SetReady(false)

	time.Sleep(time.Millisecond)

	assert.Check(t, mock01.isEmptyExpectations())
	assert.Check(t, mock02.isEmptyExpectations())
}

func TestClusterTrader_Unsubscribe(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mock01 := NewMockTrader(logger)
	mock02 := NewMockTrader(logger)

	cluster := newClusterTrader(logger, []Trader{mock01, mock02})
	assert.Equal(t, cluster.IsReady(), false)

	mock01.SetReady(true)
	mock02.SetReady(true)

	time.Sleep(time.Millisecond)

	userID := uint64(2001234)

	mock01.addExpectation(userID, expectation{GetSnapshot: true})

	orders, err := cluster.GetSnapshot(context.Background(), userID)
	assert.NilError(t, err)
	assert.Equal(t, len(orders), 0)
	assert.Check(t, mock01.isEmptyExpectations())
	assert.Check(t, mock02.isEmptyExpectations())

	mock01.addExpectation(userID, expectation{Unsubscribe: true})
	cluster.Unsubscribe(userID)

	mock02.addExpectation(userID, expectation{GetSnapshot: true})

	orders, err = cluster.GetSnapshot(context.Background(), userID)
	assert.NilError(t, err)
	assert.Equal(t, len(orders), 0)

	assert.Check(t, mock01.isEmptyExpectations())
	assert.Check(t, mock02.isEmptyExpectations())
}

func TestClusterTrader_NewOrder(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mock01 := NewMockTrader(logger)
	mock02 := NewMockTrader(logger)

	userID := uint64(2001234)

	cluster := newClusterTrader(logger, []Trader{mock01, mock02})
	assert.Equal(t, cluster.IsReady(), false)

	req := &RequestOrderNew{
		ClientOrderID: ClientOrderIdGenerateFast(123),
		Price:         "10.02",
	}

	t.Run("fail get gate", func(t *testing.T) {
		_, _, err := cluster.NewOrder(context.Background(), userID, req)
		assert.ErrorContains(t, err, "available")
	})

	mock01.SetReady(true)
	mock02.SetReady(true)

	time.Sleep(time.Millisecond)

	mock01.addExpectation(userID, expectation{
		reqId: ClientOrderIdGenerateFast(123),
		report: &OrderReport{
			OrderModel: OrderModel{
				ClientOrderID: req.ClientOrderID,
				OrderStatus:   OrderStatusNew,
				Price:         "10.02",
			},
			ReportType: ReportTypeNew,
		}})

	order, trades, err := cluster.NewOrder(context.Background(), userID, req)
	assert.NilError(t, err)
	assert.Equal(t, len(trades), 0)
	assert.Equal(t, order.ClientOrderID, req.ClientOrderID)
	assert.Equal(t, order.Price, "10.02")

	assert.Check(t, mock01.isEmptyExpectations())
	assert.Check(t, mock02.isEmptyExpectations())
}
