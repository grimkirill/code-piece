package trading

import "strconv"

type RequestOrderNew struct {
	ClientOrderID ClientOrderId    `json:"clientOrderId"`
	Type          OrderType        `json:"type"`
	Side          OrderSide        `json:"side"`
	TimeInForce   OrderTimeInForce `json:"timeInForce"`
	PostOnly      bool             `json:"participateDoNotInitiate"`
	Symbol        string           `json:"symbol"`
	Quantity      string           `json:"quantity"`
	Price         string           `json:"price,omitempty"`
	StopPrice     string           `json:"stopPrice,omitempty"`
	ExpireTime    uint64           `json:"expireTime,omitempty"`
	Login         string           `json:"login,omitempty"`
	WaitComplete  bool
}

func (r *RequestOrderNew) getPayload(userID uint64) payloadOrderNew {
	return payloadOrderNew{
		UserID:        messageUserID(userID),
		ClientOrderID: r.ClientOrderID,
		Type:          r.Type,
		Side:          r.Side,
		TimeInForce:   r.TimeInForce,
		PostOnly:      r.PostOnly,
		Symbol:        r.Symbol,
		Quantity:      r.Quantity,
		Price:         r.Price,
		StopPrice:     r.StopPrice,
		ExpireTime:    strconv.FormatUint(r.ExpireTime, 10),
	}
}

type payloadOrderNew struct {
	UserID        messageUserID    `json:"userId"`
	ClientOrderID ClientOrderId    `json:"clientOrderId"`
	Type          OrderType        `json:"type"`
	Side          OrderSide        `json:"side"`
	TimeInForce   OrderTimeInForce `json:"timeInForce"`
	PostOnly      bool             `json:"participateDoNotInitiate"`
	Symbol        string           `json:"symbol"`
	Quantity      string           `json:"quantity"`
	Price         string           `json:"price,omitempty"`
	StopPrice     string           `json:"stopPrice,omitempty"`
	ExpireTime    string           `json:"expireTime,omitempty"`
	Login         string           `json:"login,omitempty"`
}

type requestOrderCancel struct {
	UserID                     messageUserID `json:"userId"`
	ClientOrderID              ClientOrderId `json:"clientOrderId"`
	CancelRequestClientOrderID ClientOrderId `json:"cancelRequestClientOrderId"`
	Symbol                     string        `json:"symbol"`
	Side                       OrderSide     `json:"side"`
	Login                      string        `json:"login,omitempty"`
}

type requestOrderReplacePayload struct {
	UserID                     messageUserID    `json:"userId"`
	ClientOrderID              ClientOrderId    `json:"clientOrderId"`
	CancelRequestClientOrderID ClientOrderId    `json:"cancelRequestClientOrderId"`
	Symbol                     string           `json:"symbol"`
	Side                       OrderSide        `json:"side"`
	Type                       OrderType        `json:"type"`
	TimeInForce                OrderTimeInForce `json:"timeInForce"`
	Quantity                   string           `json:"quantity"`
	Price                      string           `json:"price"`
	Login                      string           `json:"login,omitempty"`
}

type RequestOrderReplace struct {
	ClientOrderID        ClientOrderId
	RequestClientOrderID ClientOrderId
	Quantity             string
	Price                string
}

type transportRequestNew struct {
	Data  payloadOrderNew `json:"NewOrder"`
	Token string          `json:"token,omitempty"`
}

type transportRequestCancel struct {
	Data  requestOrderCancel `json:"OrderCancel"`
	Token string             `json:"token,omitempty"`
}

type transportRequestReplace struct {
	Data  requestOrderReplacePayload `json:"CancelReplaceOrder"`
	Token string                     `json:"token,omitempty"`
}
