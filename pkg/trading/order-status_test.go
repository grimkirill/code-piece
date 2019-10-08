package trading_test

import (
	"testing"

	"encoding/json"

	"github.com/json-iterator/go"
	"gitlab.heather.loc/helios/astra/pkg/trading"
	"gotest.tools/assert"
)

type testOrderStatusType struct {
	Status trading.OrderStatus `json:"status"`
}

func orderStatusGetMap() map[string]trading.OrderStatus {
	return map[string]trading.OrderStatus{
		"new":             trading.OrderStatusNew,
		"suspended":       trading.OrderStatusSuspended,
		"partiallyFilled": trading.OrderStatusPartiallyFilled,
		"filled":          trading.OrderStatusFilled,
		"canceled":        trading.OrderStatusCanceled,
		"rejected":        trading.OrderStatusRejected,
		"expired":         trading.OrderStatusExpired,
	}
}

func TestOrderStatus_MarshalJSON(t *testing.T) {
	var err error
	var result []byte
	var obj testOrderStatusType

	for valStr, val := range orderStatusGetMap() {
		jsonObj := testOrderStatusType{Status: val}
		jsonStr := `{"status":"` + valStr + `"}`

		result, err = json.Marshal(&jsonObj)
		assert.NilError(t, err)
		assert.Equal(t, string(result), jsonStr, "std marshal "+valStr)

		result, err = jsoniter.Marshal(&jsonObj)
		assert.NilError(t, err)
		assert.Equal(t, string(result), jsonStr, "jsoniter marshal "+valStr)

		err = json.Unmarshal([]byte(jsonStr), &obj)
		assert.NilError(t, err)
		assert.Equal(t, obj.Status, val, "std unmarshal "+valStr)

		err = jsoniter.Unmarshal([]byte(jsonStr), &obj)
		assert.NilError(t, err)
		assert.Equal(t, obj.Status, val, "jsoniter unmarshal "+valStr)
	}

	_, err = json.Marshal(&testOrderStatusType{Status: trading.OrderStatus(100)})
	assert.ErrorContains(t, err, `invalid order status json conversion: 100`)

	_, err = jsoniter.Marshal(&testOrderStatusType{Status: trading.OrderStatus(100)})
	assert.ErrorContains(t, err, `invalid order status json conversion: 100`)

	err = json.Unmarshal([]byte(`{"status":"newStatus"}`), &obj)
	assert.ErrorContains(t, err, `unsupported order status: "newStatus"`)

	err = jsoniter.Unmarshal([]byte(`{"status":"newStatus"}`), &obj)
	assert.ErrorContains(t, err, `unsupported order status: "newStatus"`)
}

func TestOrderStatus_String(t *testing.T) {

	for valStr, val := range orderStatusGetMap() {
		assert.Equal(t, val.String(), valStr, "string "+valStr)
		resolve, err := trading.OrderStatusStrToType(valStr)
		assert.NilError(t, err)
		assert.Equal(t, resolve, val, "from string "+valStr)
	}

	defer func() {
		if r := recover(); r != nil {
		} else {
			t.Fatal("not recoverd")
		}
	}()
	_ = trading.OrderStatus(100).String()
	t.Errorf("The code did not panic")
}

func TestOrderStatus_StrToOrderTypeError(t *testing.T) {
	_, err := trading.OrderStatusStrToType("newTime")
	assert.Error(t, err, `unsupported order status: newTime`)
}
