package trading

import (
	"bytes"
	"errors"
	"strconv"
)

type OrderStatus uint8

const (
	OrderStatusNew OrderStatus = iota
	OrderStatusSuspended
	OrderStatusPartiallyFilled
	OrderStatusFilled
	OrderStatusCanceled
	OrderStatusRejected
	OrderStatusExpired

	orderStatusNewStr             = "new"
	orderStatusSuspendedStr       = "suspended"
	orderStatusPartiallyFilledStr = "partiallyFilled"
	orderStatusFilledStr          = "filled"
	orderStatusCanceledStr        = "canceled"
	orderStatusRejectedStr        = "rejected"
	orderStatusExpiredStr         = "expired"
)

var (
	orderStatusNewBytes             = []byte(`"new"`)
	orderStatusSuspendedBytes       = []byte(`"suspended"`)
	orderStatusPartiallyFilledBytes = []byte(`"partiallyFilled"`)
	orderStatusFilledBytes          = []byte(`"filled"`)
	orderStatusCanceledBytes        = []byte(`"canceled"`)
	orderStatusRejectedBytes        = []byte(`"rejected"`)
	orderStatusExpiredBytes         = []byte(`"expired"`)
)

func (os OrderStatus) String() string {

	switch os {
	case OrderStatusNew:
		return orderStatusNewStr
	case OrderStatusSuspended:
		return orderStatusSuspendedStr
	case OrderStatusPartiallyFilled:
		return orderStatusPartiallyFilledStr
	case OrderStatusFilled:
		return orderStatusFilledStr
	case OrderStatusCanceled:
		return orderStatusCanceledStr
	case OrderStatusRejected:
		return orderStatusRejectedStr
	case OrderStatusExpired:
		return orderStatusExpiredStr
	}
	panic("invalid order status string conversion" + strconv.Itoa(int(os)))
}

func (os OrderStatus) MarshalJSON() ([]byte, error) {
	switch os {
	case OrderStatusNew:
		return orderStatusNewBytes, nil
	case OrderStatusSuspended:
		return orderStatusSuspendedBytes, nil
	case OrderStatusPartiallyFilled:
		return orderStatusPartiallyFilledBytes, nil
	case OrderStatusFilled:
		return orderStatusFilledBytes, nil
	case OrderStatusCanceled:
		return orderStatusCanceledBytes, nil
	case OrderStatusRejected:
		return orderStatusRejectedBytes, nil
	case OrderStatusExpired:
		return orderStatusExpiredBytes, nil
	}
	return nil, errors.New("invalid order status json conversion: " + strconv.Itoa(int(os)))
}

func (os *OrderStatus) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, orderStatusNewBytes) {
		*os = OrderStatusNew
		return nil
	}
	if bytes.Equal(data, orderStatusCanceledBytes) {
		*os = OrderStatusCanceled
		return nil
	}
	if bytes.Equal(data, orderStatusRejectedBytes) {
		*os = OrderStatusRejected
		return nil
	}
	if bytes.Equal(data, orderStatusPartiallyFilledBytes) {
		*os = OrderStatusPartiallyFilled
		return nil
	}
	if bytes.Equal(data, orderStatusFilledBytes) {
		*os = OrderStatusFilled
		return nil
	}
	if bytes.Equal(data, orderStatusSuspendedBytes) {
		*os = OrderStatusSuspended
		return nil
	}
	if bytes.Equal(data, orderStatusExpiredBytes) {
		*os = OrderStatusExpired
		return nil
	}

	return errors.New("unsupported order status: " + string(data))
}

func OrderStatusStrToType(value string) (OrderStatus, error) {
	switch value {
	case orderStatusNewStr:
		return OrderStatusNew, nil
	case orderStatusSuspendedStr:
		return OrderStatusSuspended, nil
	case orderStatusPartiallyFilledStr:
		return OrderStatusPartiallyFilled, nil
	case orderStatusFilledStr:
		return OrderStatusFilled, nil
	case orderStatusCanceledStr:
		return OrderStatusCanceled, nil
	case orderStatusRejectedStr:
		return OrderStatusRejected, nil
	case orderStatusExpiredStr:
		return OrderStatusExpired, nil
	}
	return 0, errors.New("unsupported order status: " + value)
}
