package trading_test

import (
	"encoding/json"
	"testing"

	"github.com/json-iterator/go"
	"gitlab.heather.loc/helios/astra/pkg/trading"
	"gotest.tools/assert"
)

type testOrderDataType struct {
	Type trading.OrderType `json:"type"`
}

const (
	testOrderDataTypeMarket     = `{"type":"market"}`
	testOrderDataTypeLimit      = `{"type":"limit"}`
	testOrderDataTypeStopMarket = `{"type":"stopMarket"}`
	testOrderDataTypeStopLimit  = `{"type":"stopLimit"}`
)

func TestOrderType_MarshalJSON(t *testing.T) {
	val, err := json.Marshal(&testOrderDataType{trading.OrderTypeMarket})
	assert.NilError(t, err)
	assert.Equal(t, string(val), testOrderDataTypeMarket, "std json market")

	val, err = jsoniter.Marshal(&testOrderDataType{trading.OrderTypeMarket})
	assert.NilError(t, err)
	assert.Equal(t, string(val), testOrderDataTypeMarket, "jsoniter json market")

	val, err = json.Marshal(&testOrderDataType{trading.OrderTypeLimit})
	assert.NilError(t, err)
	assert.Equal(t, string(val), testOrderDataTypeLimit, "std json limit")

	val, err = jsoniter.Marshal(&testOrderDataType{trading.OrderTypeLimit})
	assert.NilError(t, err)
	assert.Equal(t, string(val), testOrderDataTypeLimit, "jsoniter json limit")

	val, err = json.Marshal(&testOrderDataType{trading.OrderTypeStopMarket})
	assert.NilError(t, err)
	assert.Equal(t, string(val), testOrderDataTypeStopMarket, "std json stopMarket")

	val, err = jsoniter.Marshal(&testOrderDataType{trading.OrderTypeStopMarket})
	assert.NilError(t, err)
	assert.Equal(t, string(val), testOrderDataTypeStopMarket, "jsoniter json stopMarket")

	val, err = json.Marshal(&testOrderDataType{trading.OrderTypeStopLimit})
	assert.NilError(t, err)
	assert.Equal(t, string(val), testOrderDataTypeStopLimit, "std json stopLimit")

	val, err = jsoniter.Marshal(&testOrderDataType{trading.OrderTypeStopLimit})
	assert.NilError(t, err)
	assert.Equal(t, string(val), testOrderDataTypeStopLimit, "jsoniter json stopLimit")

	_, err = json.Marshal(&testOrderDataType{trading.OrderType(8)})
	assert.ErrorContains(t, err, `invalid order type json conversion: 8`)

	_, err = jsoniter.Marshal(&testOrderDataType{trading.OrderType(8)})
	assert.ErrorContains(t, err, `invalid order type json conversion: 8`)
}

func TestOrderType_UnmarshalJSON(t *testing.T) {
	var obj testOrderDataType

	err := json.Unmarshal([]byte(testOrderDataTypeMarket), &obj)
	assert.NilError(t, err)
	assert.Equal(t, obj.Type, trading.OrderTypeMarket, "std json market")

	err = jsoniter.Unmarshal([]byte(testOrderDataTypeMarket), &obj)
	assert.NilError(t, err)
	assert.Equal(t, obj.Type, trading.OrderTypeMarket, "jsoniter json market")

	err = json.Unmarshal([]byte(testOrderDataTypeLimit), &obj)
	assert.NilError(t, err)
	assert.Equal(t, obj.Type, trading.OrderTypeLimit, "std json limit")

	err = jsoniter.Unmarshal([]byte(testOrderDataTypeLimit), &obj)
	assert.NilError(t, err)
	assert.Equal(t, obj.Type, trading.OrderTypeLimit, "jsoniter json limit")

	err = json.Unmarshal([]byte(testOrderDataTypeStopMarket), &obj)
	assert.NilError(t, err)
	assert.Equal(t, obj.Type, trading.OrderTypeStopMarket, "std json stopMarket")

	err = jsoniter.Unmarshal([]byte(testOrderDataTypeStopMarket), &obj)
	assert.NilError(t, err)
	assert.Equal(t, obj.Type, trading.OrderTypeStopMarket, "jsoniter json stopMarket")

	err = json.Unmarshal([]byte(testOrderDataTypeStopLimit), &obj)
	assert.NilError(t, err)
	assert.Equal(t, obj.Type, trading.OrderTypeStopLimit, "std json stopLimit")

	err = jsoniter.Unmarshal([]byte(testOrderDataTypeStopLimit), &obj)
	assert.NilError(t, err)
	assert.Equal(t, obj.Type, trading.OrderTypeStopLimit, "jsoniter json stopLimit")

	err = json.Unmarshal([]byte(`{"type":"newType"}`), &obj)
	assert.Error(t, err, `unsupported order type: "newType"`)
}

func TestOrderType_String(t *testing.T) {
	valid := map[string]trading.OrderType{
		"market":     trading.OrderTypeMarket,
		"limit":      trading.OrderTypeLimit,
		"stopMarket": trading.OrderTypeStopMarket,
		"stopLimit":  trading.OrderTypeStopLimit,
	}
	for valStr, val := range valid {
		assert.Equal(t, val.String(), valStr, "string "+valStr)
		resolve, err := trading.OrderTypeStrToType(valStr)
		assert.NilError(t, err)
		assert.Equal(t, resolve, val, "from string "+valStr)
	}

	defer func() {
		if r := recover(); r != nil {
		} else {
			t.Fatal("not recoverd")
		}
	}()
	_ = trading.OrderType(5).String()
	t.Errorf("The code did not panic")
}

func TestOrderType_StrToOrderTypeError(t *testing.T) {
	_, err := trading.OrderTypeStrToType("newType")
	assert.Error(t, err, `unsupported order type: newType`)
}
