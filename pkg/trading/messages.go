package trading

import (
	"encoding/json"
	"strconv"

	"bytes"

	"github.com/pkg/errors"
)

type messageUserID uint64

func (u messageUserID) UInt64() uint64 {
	return uint64(u)
}

var userJsonPrefix = []byte(`"user_`)

func (u messageUserID) MarshalJSON() ([]byte, error) {
	idBytes := []byte(strconv.FormatUint(uint64(u), 10))
	result := make([]byte, len(idBytes)+7)
	copy(result, userJsonPrefix)
	copy(result[6:], idBytes)
	result[len(result)-1] = '"'
	return result, nil
}

func (u *messageUserID) UnmarshalJSON(data []byte) error {
	if !bytes.HasPrefix(data, userJsonPrefix) {
		return errors.New("unsupported user field: should contain prefix user_, got: " + string(data))
	}
	id, err := strconv.ParseUint(string(data[6:len(data)-1]), 10, 64)
	if err != nil {
		return errors.New("unsupported user field: invalid number, got: " + string(data) + " err: " + err.Error())
	}
	*u = messageUserID(id)
	return nil
}

type messageSnapshot struct {
	Data  messageSnapshotContent `json:"Snapshot"`
	Token string                 `json:"token,omitempty"`
}

type messageSnapshotContent struct {
	User   string                 `json:"user"`
	Orders []messageSnapshotOrder `json:"orders"`
}

type messageSnapshotOrder struct {
	OrderID        string           `json:"orderId"`
	LatestOrderID  string           `json:"latestOrderId"`
	ClientOrderID  ClientOrderId    `json:"clientOrderId"`
	OrderStatus    OrderStatus      `json:"orderStatus"`
	PostOnly       bool             `json:"participateDoNotInitiate"`
	Symbol         string           `json:"symbol"`
	Side           OrderSide        `json:"side"`
	Price          string           `json:"price"`
	Quantity       uint64           `json:"quantity"`
	Type           OrderType        `json:"type"`
	TimeInForce    OrderTimeInForce `json:"timeInForce"`
	LastQuantity   uint64           `json:"lastQuantity"`
	LastPrice      string           `json:"lastPrice"`
	LeavesQuantity uint64           `json:"leavesQuantity"`
	CumQuantity    json.Number      `json:"cumQuantity"`
	AveragePrice   string           `json:"averagePrice"`
	Created        uint64           `json:"created"`
}

func (o *messageSnapshotOrder) CreateOrderModel() (order OrderModel) {
	order.ClientOrderID = o.ClientOrderID
	order.OrderStatus = o.OrderStatus
	order.Symbol = o.Symbol
	order.Quantity = strconv.FormatUint(o.Quantity, 10)
	order.CumQuantity = string(o.CumQuantity)
	order.LeavesQuantity = o.LeavesQuantity
	order.Price = o.Price
	order.LastQuantity = o.LastQuantity
	order.LastPrice = o.LastPrice
	order.Created = o.Created
	order.Timestamp = o.Created
	order.Side = o.Side
	order.PostOnly = o.PostOnly
	order.Type = o.Type
	order.TimeInForce = o.TimeInForce

	var err error
	order.OrderID, err = strconv.ParseUint(o.OrderID, 10, 64)
	if err != nil {
		panic(errors.WithMessage(err, "zmq: fail parse orderId "+o.OrderID))
	}
	order.LatestOrderID, err = strconv.ParseUint(o.LatestOrderID, 10, 64)
	if err != nil {
		panic(errors.WithMessage(err, "zmq: fail parse LatestOrderID "+o.LatestOrderID))
	}
	return
}

type messageReport struct {
	Data  messageOrderReport `json:"ExecutionReport"`
	Token string             `json:"token,omitempty"`
}

type messageOrderReport struct {
	OrderID               string           `json:"orderId"`
	LastOrderID           string           `json:"latestOrderId"`
	ClientOrderID         ClientOrderId    `json:"clientOrderId"`
	OriginalClientOrderID ClientOrderId    `json:"originalRequestClientOrderId"`
	OrderStatus           OrderStatus      `json:"orderStatus"`
	Side                  OrderSide        `json:"side"`
	Type                  OrderType        `json:"type"`
	TimeInForce           OrderTimeInForce `json:"timeInForce"`
	ExecReportType        ReportType       `json:"execReportType"`
	PostOnly              bool             `json:"participateDoNotInitiate"`
	UserID                string           `json:"userId"`
	Symbol                string           `json:"symbol"`
	Price                 string           `json:"price"`
	AveragePrice          string           `json:"averagePrice"`
	LastPrice             string           `json:"lastPrice"`
	Quantity              uint64           `json:"quantity"`
	LastQuantity          uint64           `json:"lastQuantity"`
	LeavesQuantity        uint64           `json:"leavesQuantity"`
	CumQuantity           json.Number      `json:"cumQuantity"`
	Created               uint64           `json:"created"`
	Timestamp             uint64           `json:"timestamp"`
	OrderRejectReason     string           `json:"orderRejectReason"`
	TradeID               uint64           `json:"tradeId"`
	FeeAmount             string           `json:"feeAmount"`
	FeeInstrument         string           `json:"feeInstr"`
}

func (m *messageOrderReport) CreateOrderReport() (report OrderReport) {
	report.OrderModel = m.CreateOrderModel()
	report.ReportType = m.ExecReportType
	if m.ExecReportType == ReportTypeReplaced {
		report.OriginalClientOrderID = m.OriginalClientOrderID
	}
	return
}

func (m *messageOrderReport) CreateOrderModel() (order OrderModel) {
	order.ClientOrderID = m.ClientOrderID
	order.OrderStatus = m.OrderStatus
	order.Symbol = m.Symbol
	order.Quantity = strconv.FormatUint(m.Quantity, 10)
	order.CumQuantity = string(m.CumQuantity)
	order.LeavesQuantity = m.LeavesQuantity
	order.Price = m.Price
	order.LastQuantity = m.LastQuantity
	order.LastPrice = m.LastPrice
	order.Created = m.Created
	order.Timestamp = m.Timestamp
	order.Side = m.Side
	order.PostOnly = m.PostOnly
	order.Type = m.Type
	order.TimeInForce = m.TimeInForce

	var err error
	order.OrderID, err = strconv.ParseUint(m.OrderID, 10, 64)
	if err != nil {
		panic(errors.WithMessage(err, "zmq: fail parse orderId "+m.OrderID))
	}
	order.LatestOrderID, err = strconv.ParseUint(m.LastOrderID, 10, 64)
	if err != nil {
		panic(errors.WithMessage(err, "zmq: fail parse LatestOrderID "+m.LastOrderID))
	}
	return
}

type messageCancelContent struct {
	CancelReject messageCancelReject `json:"CancelReject"`
	Token        string              `json:"token,omitempty"`
}

type messageCancelReplaceContent struct {
	CancelReject messageCancelReject `json:"CancelReplaceReject"`
	Token        string              `json:"token,omitempty"`
}

type messageCancelReject struct {
	UserID                     string        `json:"userId"`
	CancelRequestClientOrderID ClientOrderId `json:"cancelRequestClientOrderId"`
	ClientOrderID              ClientOrderId `json:"clientOrderId"`
	RejectReasonCode           string        `json:"rejectReasonCode"`
}

func userStrToID(user string) uint64 {
	id, err := strconv.ParseUint(user[5:], 10, 64)
	if err != nil {
		panic(err)
	}
	return id
}

type subscribeReplyContent struct {
	Data  subscribtionReply `json:"SubscriptionReply"`
	Token string            `json:"token,omitempty"`
}

type subscribtionReply struct {
	ResponseCode string `json:"responseCode"`
	UserID       string `json:"userId"`
}
