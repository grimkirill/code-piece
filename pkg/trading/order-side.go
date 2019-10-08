package trading

import (
	"bytes"
	"errors"
	"strconv"
)

type OrderSide uint8

const (
	OrderSideSell OrderSide = iota
	OrderSideBuy

	orderSideSellStr = "sell"
	orderSideBuyStr  = "buy"
)

var (
	orderSideSellByte = []byte(`"sell"`)
	orderSideBuyByte  = []byte(`"buy"`)
)

func (ot OrderSide) String() string {
	switch ot {
	case OrderSideSell:
		return orderSideSellStr
	case OrderSideBuy:
		return orderSideBuyStr
	}
	panic("invalid order side string conversion" + strconv.Itoa(int(ot)))
}

func (ot OrderSide) MarshalJSON() ([]byte, error) {
	switch ot {
	case OrderSideSell:
		return orderSideSellByte, nil
	case OrderSideBuy:
		return orderSideBuyByte, nil
	}
	return nil, errors.New("invalid order side json conversion: " + strconv.Itoa(int(ot)))
}

func (ot *OrderSide) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, orderSideSellByte) {
		*ot = OrderSideSell
		return nil
	}

	if bytes.Equal(data, orderSideBuyByte) {
		*ot = OrderSideBuy
		return nil
	}

	return errors.New("unsupported order side: " + string(data))
}

func OrderSideStrToType(value string) (OrderSide, error) {
	switch value {
	case orderSideSellStr:
		return OrderSideSell, nil
	case orderSideBuyStr:
		return OrderSideBuy, nil
	}
	return 0, errors.New("unsupported order side: " + value)
}
