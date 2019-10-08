package trading

import "context"

type Trader interface {
	Unsubscribe(userID uint64)
	GetSnapshot(ctx context.Context, userID uint64) ([]OrderModel, error)
	GetOrder(ctx context.Context, userID uint64, clientOrderID ClientOrderId) (*OrderModel, error)
	//NewOrder(ctx context.Context, userID uint64, order *RequestOrderNew) (*OrderReport, error)
	NewOrder(ctx context.Context, userID uint64, order *RequestOrderNew) (*OrderReport, []Trade, error)
	CancelOrder(ctx context.Context, userID uint64, clientOrderID ClientOrderId) (*OrderReport, error)
	ReplaceOrder(ctx context.Context, userID uint64, req RequestOrderReplace) (*OrderReport, error)
	IsReady() bool
	Ready() chan bool
}
