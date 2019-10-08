package trading

import (
	"bytes"
	"errors"
	"strconv"
)

type OrderTimeInForce uint8

const (
	OrderTimeInForceGTC OrderTimeInForce = iota // good till cancel. cancel only by user request of full filled
	OrderTimeInForceIOC                         // immediate or cancel. can be partially filled and then final status expired
	OrderTimeInForceFOK                         // fill or kill. fully filled or expired
	OrderTimeInForceDAY                         // Day. automatically expired at the end of the day (00 UTC)
	OrderTimeInForceGTD                         // good till date. automatically expired on a specified time

	orderTimeInForceGTCstr = "GTC" // good till cancel. cancel only by user request of full filled
	orderTimeInForceIOCstr = "IOC" // immediate or cancel. can be partially filled and then final status expired
	orderTimeInForceFOKstr = "FOK" // fill or kill. fully filled or expired
	orderTimeInForceDAYstr = "Day" // Day. automatically expired at the end of the day (00 UTC)
	orderTimeInForceGTDstr = "GTD" // good till date. automatically expired on a specified time
)

var (
	orderTimeInForceGTCbytes = []byte(`"GTC"`)
	orderTimeInForceIOCbytes = []byte(`"IOC"`)
	orderTimeInForceFOKbytes = []byte(`"FOK"`)
	orderTimeInForceDAYbytes = []byte(`"Day"`)
	orderTimeInForceGTDbytes = []byte(`"GTD"`)
)

func (tif OrderTimeInForce) String() string {
	switch tif {
	case OrderTimeInForceGTC:
		return orderTimeInForceGTCstr
	case OrderTimeInForceIOC:
		return orderTimeInForceIOCstr
	case OrderTimeInForceFOK:
		return orderTimeInForceFOKstr
	case OrderTimeInForceDAY:
		return orderTimeInForceDAYstr
	case OrderTimeInForceGTD:
		return orderTimeInForceGTDstr
	}
	panic("invalid order timeInForce string conversion" + strconv.Itoa(int(tif)))
}

func (tif OrderTimeInForce) MarshalJSON() ([]byte, error) {
	switch tif {
	case OrderTimeInForceGTC:
		return orderTimeInForceGTCbytes, nil
	case OrderTimeInForceIOC:
		return orderTimeInForceIOCbytes, nil
	case OrderTimeInForceFOK:
		return orderTimeInForceFOKbytes, nil
	case OrderTimeInForceDAY:
		return orderTimeInForceDAYbytes, nil
	case OrderTimeInForceGTD:
		return orderTimeInForceGTDbytes, nil
	}
	return nil, errors.New("invalid order timeInForce json conversion: " + strconv.Itoa(int(tif)))
}

func (tif *OrderTimeInForce) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, orderTimeInForceGTCbytes) {
		*tif = OrderTimeInForceGTC
		return nil
	}

	if bytes.Equal(data, orderTimeInForceIOCbytes) {
		*tif = OrderTimeInForceIOC
		return nil
	}

	if bytes.Equal(data, orderTimeInForceFOKbytes) {
		*tif = OrderTimeInForceFOK
		return nil
	}

	if bytes.Equal(data, orderTimeInForceDAYbytes) {
		*tif = OrderTimeInForceDAY
		return nil
	}

	if bytes.Equal(data, orderTimeInForceGTDbytes) {
		*tif = OrderTimeInForceGTD
		return nil
	}

	return errors.New("unsupported order timeInForce: " + string(data))
}

func OrderTimeInForceStrToType(value string) (OrderTimeInForce, error) {
	switch value {
	case orderTimeInForceGTCstr:
		return OrderTimeInForceGTC, nil
	case orderTimeInForceFOKstr:
		return OrderTimeInForceFOK, nil
	case orderTimeInForceIOCstr:
		return OrderTimeInForceIOC, nil
	case orderTimeInForceDAYstr:
		return OrderTimeInForceDAY, nil
	case orderTimeInForceGTDstr:
		return OrderTimeInForceGTD, nil

	}
	return 0, errors.New("unsupported order timeInForce: " + value)
}
