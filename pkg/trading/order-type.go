package trading

import (
	"bytes"
	"errors"
	"strconv"
)

type OrderType uint8

const (
	OrderTypeMarket OrderType = iota
	OrderTypeLimit
	OrderTypeStopMarket
	OrderTypeStopLimit

	orderTypeMarketStr     = "market"
	orderTypeLimitStr      = "limit"
	orderTypeStopMarketStr = "stopMarket"
	orderTypeStopLimitStr  = "stopLimit"
)

var (
	orderTypeMarketByte     = []byte(`"market"`)
	orderTypeLimitByte      = []byte(`"limit"`)
	orderTypeStopMarketByte = []byte(`"stopMarket"`)
	orderTypeStopLimitByte  = []byte(`"stopLimit"`)
)

func (ot OrderType) String() string {
	switch ot {
	case OrderTypeMarket:
		return orderTypeMarketStr
	case OrderTypeLimit:
		return orderTypeLimitStr
	case OrderTypeStopMarket:
		return orderTypeStopMarketStr
	case OrderTypeStopLimit:
		return orderTypeStopLimitStr
	}
	panic("invalid order type string conversion" + strconv.Itoa(int(ot)))
}

func (ot OrderType) MarshalJSON() ([]byte, error) {
	switch ot {
	case OrderTypeMarket:
		return orderTypeMarketByte, nil
	case OrderTypeLimit:
		return orderTypeLimitByte, nil
	case OrderTypeStopMarket:
		return orderTypeStopMarketByte, nil
	case OrderTypeStopLimit:
		return orderTypeStopLimitByte, nil
	}
	return nil, errors.New("invalid order type json conversion: " + strconv.Itoa(int(ot)))
}

func (ot *OrderType) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, orderTypeMarketByte) {
		*ot = OrderTypeMarket
		return nil
	}

	if bytes.Equal(data, orderTypeLimitByte) {
		*ot = OrderTypeLimit
		return nil
	}

	if bytes.Equal(data, orderTypeStopMarketByte) {
		*ot = OrderTypeStopMarket
		return nil
	}

	if bytes.Equal(data, orderTypeStopLimitByte) {
		*ot = OrderTypeStopLimit
		return nil
	}

	return errors.New("unsupported order type: " + string(data))
}

func OrderTypeStrToType(value string) (OrderType, error) {
	switch value {
	case orderTypeMarketStr:
		return OrderTypeMarket, nil
	case orderTypeLimitStr:
		return OrderTypeLimit, nil
	case orderTypeStopMarketStr:
		return OrderTypeStopMarket, nil
	case orderTypeStopLimitStr:
		return OrderTypeStopLimit, nil
	}
	return 0, errors.New("unsupported order type: " + value)
}
