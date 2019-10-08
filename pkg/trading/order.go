package trading

type OrderModel struct {
	OrderID        uint64
	LatestOrderID  uint64
	ClientOrderID  ClientOrderId
	Type           OrderType
	TimeInForce    OrderTimeInForce
	Side           OrderSide
	OrderStatus    OrderStatus
	PostOnly       bool
	Symbol         string
	Price          string
	StopPrice      string
	ExpireTime     uint64
	Quantity       string
	LastQuantity   uint64
	LastPrice      string
	LeavesQuantity uint64
	CumQuantity    string
	AveragePrice   string
	Created        uint64
	Timestamp      uint64
}

type Trade struct {
	ID            uint64
	Timestamp     uint64
	Quantity      uint64
	Price         string
	FeeAmount     string
	FeeInstrument string
}

type OrderReport struct {
	OrderModel
	ReportType            ReportType
	OriginalClientOrderID ClientOrderId
	Timestamp             uint64
	TradeID               uint64
	FeeAmount             string
	FeeInstrument         string
}

type payloadSnapshot struct {
	userID uint64
	orders []OrderModel
}

type payloadReport struct {
	userID uint64
	report OrderReport
}

type payloadReject struct {
	userID                     uint64
	ClientOrderID              ClientOrderId
	CancelRequestClientOrderID ClientOrderId
	Timestamp                  uint64
	RejectReason               string
}
