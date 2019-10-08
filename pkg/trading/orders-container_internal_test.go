package trading

import (
	"bytes"
	"fmt"
	"runtime"
	"testing"
	"time"

	"gotest.tools/assert"
)

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func TestOrdersContainerFlow(t *testing.T) {
	orderA := OrderModel{
		ClientOrderID: ClientOrderIdGenerateFast(12345),
		OrderStatus:   OrderStatusNew,
	}
	orderB := OrderModel{
		ClientOrderID: ClientOrderIdGenerateFast(12346),
		OrderStatus:   OrderStatusNew,
	}

	userID := uint64(1234567)

	t.Run("test check, set, delete", func(t *testing.T) {
		container := newOrderContainer()

		ok := container.hasOrders(userID)
		assert.Check(t, !ok, "no exist orders list")
		_, ok = container.getOrders(userID)
		assert.Check(t, !ok, "no orders list")

		_, ok = container.getOrder(userID, orderA.ClientOrderID)
		assert.Check(t, !ok, "no user no order")

		container.setOrders(userID, []OrderModel{})
		ok = container.hasOrders(userID)
		assert.Check(t, ok, "exist orders list")
		orders, ok := container.getOrders(userID)
		assert.Check(t, ok, "orders list not empty")
		assert.Equal(t, len(orders), 0, "orders length")

		_, ok = container.getOrder(userID, orderA.ClientOrderID)
		assert.Check(t, !ok, "no exist order in empty list")

		container.setOrders(userID, []OrderModel{orderA, orderB})

		ok = container.hasOrders(userID)
		assert.Check(t, ok, "exist orders list")
		orders, ok = container.getOrders(userID)
		assert.Check(t, ok, "orders list not empty")
		assert.Equal(t, len(orders), 2, "orders length")

		order, ok := container.getOrder(userID, orderA.ClientOrderID)
		assert.Check(t, ok, "exist order in list")
		assert.Equal(t, orderA.ClientOrderID, order.ClientOrderID, "orders length")

		container.delete(userID)
		ok = container.hasOrders(userID)
		assert.Check(t, !ok, "no longer exist orders list")
		_, ok = container.getOrders(userID)
		assert.Check(t, !ok, "no longer orders list")

		//secondary delete
		container.delete(userID)
		ok = container.hasOrders(userID)
		assert.Check(t, !ok, "no longer exist orders list")
	})

	t.Run("check handle report partially fill", func(t *testing.T) {
		container := newOrderContainer()
		container.setOrders(userID, []OrderModel{orderA, orderB})
		reportA := OrderReport{OrderModel: orderA, ReportType: ReportTypeTrade}
		reportA.OrderStatus = OrderStatusPartiallyFilled
		container.handleReport(userID, reportA)

		orders, ok := container.getOrders(userID)
		assert.Check(t, ok, "orders list not empty")
		assert.Equal(t, len(orders), 2, "orders length")
	})

	t.Run("check handle report new", func(t *testing.T) {
		container := newOrderContainer()
		container.setOrders(userID, []OrderModel{orderA, orderB})
		report := OrderReport{OrderModel: OrderModel{
			ClientOrderID: ClientOrderIdGenerateFast(12347),
			OrderStatus:   OrderStatusNew,
		}, ReportType: ReportTypeNew}

		container.handleReport(userID, report)

		orders, ok := container.getOrders(userID)
		assert.Check(t, ok, "orders list not empty")
		assert.Equal(t, len(orders), 3, "orders length")
	})

	t.Run("check handle report new suspended", func(t *testing.T) {
		container := newOrderContainer()
		container.setOrders(userID, []OrderModel{orderA, orderB})
		report := OrderReport{OrderModel: OrderModel{
			ClientOrderID: ClientOrderIdGenerateFast(12347),
			OrderStatus:   OrderStatusSuspended,
		}, ReportType: ReportTypeSuspended}

		container.handleReport(userID, report)

		orders, ok := container.getOrders(userID)
		assert.Check(t, ok, "orders list not empty")
		assert.Equal(t, len(orders), 3, "orders length")
	})

	t.Run("check handle report cancel", func(t *testing.T) {
		container := newOrderContainer()
		container.setOrders(userID, []OrderModel{orderA, orderB})
		reportA := OrderReport{OrderModel: orderA, ReportType: ReportTypeCanceled}
		reportA.OrderStatus = OrderStatusCanceled
		container.handleReport(userID, reportA)

		orders, ok := container.getOrders(userID)
		assert.Check(t, ok, "orders list not empty")
		assert.Equal(t, len(orders), 1, "orders length")
		assert.Equal(t, orders[0], orderB, "orders length")
	})

	t.Run("check handle report filled", func(t *testing.T) {
		container := newOrderContainer()
		container.setOrders(userID, []OrderModel{orderA, orderB})
		reportA := OrderReport{OrderModel: orderA, ReportType: ReportTypeTrade}
		reportA.OrderStatus = OrderStatusFilled
		container.handleReport(userID, reportA)

		orders, ok := container.getOrders(userID)
		assert.Check(t, ok, "orders list not empty")
		assert.Equal(t, len(orders), 1, "orders length")
		assert.Equal(t, orders[0], orderB, "orders length")
	})

	t.Run("check handle report expired", func(t *testing.T) {
		container := newOrderContainer()
		container.setOrders(userID, []OrderModel{orderA, orderB})
		reportA := OrderReport{OrderModel: orderA, ReportType: ReportTypeExpired}
		reportA.OrderStatus = OrderStatusExpired
		container.handleReport(userID, reportA)

		orders, ok := container.getOrders(userID)
		assert.Check(t, ok, "orders list not empty")
		assert.Equal(t, len(orders), 1, "orders length")
		assert.Equal(t, orders[0], orderB, "orders length")
	})

	t.Run("check handle report replaced", func(t *testing.T) {
		container := newOrderContainer()
		container.setOrders(userID, []OrderModel{orderA, orderB})
		report := OrderReport{OrderModel: OrderModel{
			ClientOrderID: ClientOrderIdGenerateFast(12347),
			OrderStatus:   OrderStatusNew,
		}, ReportType: ReportTypeReplaced, OriginalClientOrderID: orderB.ClientOrderID}
		container.handleReport(userID, report)

		orders, ok := container.getOrders(userID)
		assert.Check(t, ok, "orders list not empty")
		assert.Equal(t, len(orders), 2, "orders length")

		orderCheckA, _ := container.getOrder(userID, orderA.ClientOrderID)
		assert.Equal(t, orderCheckA, orderA)
		orderCheckNew, _ := container.getOrder(userID, report.ClientOrderID)
		assert.Equal(t, orderCheckNew, report.OrderModel)
	})

	t.Run("check handle report rejected", func(t *testing.T) {
		container := newOrderContainer()
		container.setOrders(userID, []OrderModel{orderA, orderB})
		report := OrderReport{OrderModel: OrderModel{
			ClientOrderID: ClientOrderIdGenerateFast(12347),
			OrderStatus:   OrderStatusRejected,
		}, ReportType: ReportTypeRejected}

		container.handleReport(userID, report)

		orders, ok := container.getOrders(userID)
		assert.Check(t, ok, "orders list not empty")
		assert.Equal(t, len(orders), 2, "orders length")
	})

	t.Run("check handle report not intilazed", func(t *testing.T) {
		container := newOrderContainer()

		report := OrderReport{OrderModel: OrderModel{
			ClientOrderID: ClientOrderIdGenerateFast(12347),
			OrderStatus:   OrderStatusNew,
		}, ReportType: ReportTypeNew}

		container.handleReport(userID, report)

		ok := container.hasOrders(userID)
		assert.Check(t, !ok, "orders list should be empty")
	})
}

func TestOrdersGCContainer(t *testing.T) {
	container := newOrderContainer()

	for i := 0; i < 10; i++ {
		orders := make([]OrderModel, 1e5)
		for j := 0; j < len(orders); j++ {
			orders[j].ClientOrderID = ClientOrderIdGenerateFast(j)
		}
		container.setOrders(uint64(i), orders)
	}

	for i := 1e6; i < 1e6+5e4; i++ {
		orders := make([]OrderModel, 10)
		for j := 0; j < len(orders); j++ {
			orders[j] = OrderModel{
				ClientOrderID: ClientOrderIdGenerateFast(j),
			}
		}
		container.setOrders(uint64(i), orders)
	}

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	fmt.Println("HeapObjects", stats.HeapObjects)
	fmt.Println("PauseTotalNs", time.Duration(int64(stats.PauseTotalNs)))
	fmt.Printf("Alloc = %v MiB", bToMb(stats.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(stats.TotalAlloc))
	fmt.Printf("\tHeapSys = %v MiB", bToMb(stats.HeapSys))
	fmt.Printf("\tHeapAlloc = %v MiB", bToMb(stats.HeapAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(stats.Sys))
	fmt.Printf("\tNumGC = %v\n", stats.NumGC)

	runtime.GC()

	for i := 0; i < 1e5; i++ {
		container.getOrder(1, ClientOrderIdGenerateFast(i))
	}

	for i := 1e6; i < 1e6+1e4; i++ {
		for j := 0; j < 10; j++ {
			container.getOrder(uint64(i), ClientOrderIdGenerateFast(j))
		}
	}
	beforeGC := time.Now()
	runtime.GC()
	gcDuration := time.Since(beforeGC)
	fmt.Println("gc duration", gcDuration)

	assert.Check(t, gcDuration < time.Millisecond*100, "gc duration time")

}

func BenchmarkOrdersContainer_GetOrder(b *testing.B) {
	container := newOrderContainer()
	userID := uint64(1)
	orders := make([]OrderModel, 1e6)
	for i := 0; i < len(orders); i++ {
		orders[i] = OrderModel{
			ClientOrderID: ClientOrderIdGenerateFast(i),
		}
	}
	container.setOrders(userID, orders)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		id := ClientOrderIdGenerateFast(i)
		order, ok := container.getOrder(userID, id)
		if ok && !bytes.Equal(order.ClientOrderID[:], id[:]) {
			b.Fatal("fail orders data match")
		}
	}
}

func BenchmarkOrdersContainer_GetOrders100(b *testing.B) {
	container := newOrderContainer()
	userID := uint64(1)
	orders := make([]OrderModel, 1e2)
	for i := 0; i < len(orders); i++ {
		orders[i] = OrderModel{
			ClientOrderID: ClientOrderIdGenerateFast(i),
		}
	}
	container.setOrders(userID, orders)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		orders, ok := container.getOrders(userID)
		if ok && len(orders) != 100 {
			b.Fatal("fail orders data match")
		}
	}
}

func BenchmarkOrdersContainer_GetOrders1000(b *testing.B) {
	container := newOrderContainer()
	userID := uint64(1)
	orders := make([]OrderModel, 1e3)
	for i := 0; i < len(orders); i++ {
		orders[i] = OrderModel{
			ClientOrderID: ClientOrderIdGenerateFast(i),
		}
	}
	container.setOrders(userID, orders)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		orders, ok := container.getOrders(userID)
		if ok && len(orders) != 1e3 {
			b.Fatal("fail orders data match")
		}
	}
}
