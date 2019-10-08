package trading_test

import (
	"encoding/json"
	"testing"

	"github.com/json-iterator/go"
	"gitlab.heather.loc/helios/astra/pkg/trading"
	"gotest.tools/assert"
)

type testOrderDataSide struct {
	Side trading.OrderSide `json:"side"`
}

const (
	testOrderDataSideSell = `{"side":"sell"}`
	testOrderDataSideBuy  = `{"side":"buy"}`
)

func TestOrderSide_MarshalJSON(t *testing.T) {
	val, err := json.Marshal(&testOrderDataSide{trading.OrderSideSell})
	assert.NilError(t, err)
	assert.Equal(t, string(val), testOrderDataSideSell, "std json sell")

	val, err = jsoniter.Marshal(&testOrderDataSide{trading.OrderSideSell})
	assert.NilError(t, err)
	assert.Equal(t, string(val), testOrderDataSideSell, "jsoniter json sell")

	val, err = json.Marshal(&testOrderDataSide{trading.OrderSideBuy})
	assert.NilError(t, err)
	assert.Equal(t, string(val), testOrderDataSideBuy, "std json buy")

	val, err = jsoniter.Marshal(&testOrderDataSide{trading.OrderSideBuy})
	assert.NilError(t, err)
	assert.Equal(t, string(val), testOrderDataSideBuy, "jsoniter json buy")

	_, err = json.Marshal(&testOrderDataSide{trading.OrderSide(8)})
	assert.ErrorContains(t, err, `invalid order side json conversion: 8`)

	_, err = jsoniter.Marshal(&testOrderDataSide{trading.OrderSide(8)})
	assert.ErrorContains(t, err, `invalid order side json conversion: 8`)
}

func TestOrderSide_UnmarshalJSON(t *testing.T) {

	var obj testOrderDataSide

	err := json.Unmarshal([]byte(testOrderDataSideSell), &obj)
	assert.NilError(t, err)
	assert.Equal(t, obj.Side, trading.OrderSideSell, "std json sell")

	err = jsoniter.Unmarshal([]byte(testOrderDataSideSell), &obj)
	assert.NilError(t, err)
	assert.Equal(t, obj.Side, trading.OrderSideSell, "jsoniter json sell")

	err = json.Unmarshal([]byte(testOrderDataSideBuy), &obj)
	assert.NilError(t, err)
	assert.Equal(t, obj.Side, trading.OrderSideBuy, "std json buy")

	err = jsoniter.Unmarshal([]byte(testOrderDataSideBuy), &obj)
	assert.NilError(t, err)
	assert.Equal(t, obj.Side, trading.OrderSideBuy, "jsoniter json buy")

	err = json.Unmarshal([]byte(`{"side":"newSide"}`), &obj)
	assert.Error(t, err, `unsupported order side: "newSide"`)
}

func TestOrderSide_String(t *testing.T) {
	assert.Equal(t, trading.OrderSideSell.String(), "sell")
	assert.Equal(t, trading.OrderSideBuy.String(), "buy")

	defer func() {
		if r := recover(); r != nil {
		} else {
			t.Fatal("not recoverd")
		}
	}()
	assert.Equal(t, trading.OrderSide(2).String(), "none")
	t.Errorf("The code did not panic")
}

func TestOrderSide_StrToType(t *testing.T) {
	resolve, err := trading.OrderSideStrToType("sell")
	assert.NilError(t, err)
	assert.Equal(t, resolve, trading.OrderSideSell, "from string sell")
	resolve, err = trading.OrderSideStrToType("buy")
	assert.NilError(t, err)
	assert.Equal(t, resolve, trading.OrderSideBuy, "from string buy")
}

func TestOrderSide_StrToTypeError(t *testing.T) {
	_, err := trading.OrderSideStrToType("newSide")
	assert.Error(t, err, `unsupported order side: newSide`)
}
