package trading_test

import (
	"testing"

	"encoding/json"

	"github.com/json-iterator/go"
	"gitlab.heather.loc/helios/astra/pkg/trading"
	"gotest.tools/assert"
)

type testOrderTimeInForceType struct {
	TimeInForce trading.OrderTimeInForce `json:"timeInForce"`
}

func timeInForceGetMap() map[string]trading.OrderTimeInForce {
	return map[string]trading.OrderTimeInForce{
		"GTC": trading.OrderTimeInForceGTC,
		"FOK": trading.OrderTimeInForceFOK,
		"IOC": trading.OrderTimeInForceIOC,
		"Day": trading.OrderTimeInForceDAY,
		"GTD": trading.OrderTimeInForceGTD,
	}
}

func TestOrderTimeInForce_MarshalJSON(t *testing.T) {
	var err error
	var result []byte
	var obj testOrderTimeInForceType

	for valStr, val := range timeInForceGetMap() {
		jsonObj := testOrderTimeInForceType{TimeInForce: val}
		jsonStr := `{"timeInForce":"` + valStr + `"}`

		result, err = json.Marshal(&jsonObj)
		assert.NilError(t, err)
		assert.Equal(t, string(result), jsonStr, "std marshal "+valStr)

		result, err = jsoniter.Marshal(&jsonObj)
		assert.NilError(t, err)
		assert.Equal(t, string(result), jsonStr, "jsoniter marshal "+valStr)

		err = json.Unmarshal([]byte(jsonStr), &obj)
		assert.NilError(t, err)
		assert.Equal(t, obj.TimeInForce, val, "std unmarshal "+valStr)

		err = jsoniter.Unmarshal([]byte(jsonStr), &obj)
		assert.NilError(t, err)
		assert.Equal(t, obj.TimeInForce, val, "jsoniter unmarshal "+valStr)
	}

	_, err = json.Marshal(&testOrderTimeInForceType{TimeInForce: trading.OrderTimeInForce(100)})
	assert.ErrorContains(t, err, `invalid order timeInForce json conversion: 100`)

	_, err = jsoniter.Marshal(&testOrderTimeInForceType{TimeInForce: trading.OrderTimeInForce(100)})
	assert.ErrorContains(t, err, `invalid order timeInForce json conversion: 100`)

	err = json.Unmarshal([]byte(`{"timeInForce":"newTimeInForce"}`), &obj)
	assert.ErrorContains(t, err, `unsupported order timeInForce: "newTimeInForce"`)

	err = jsoniter.Unmarshal([]byte(`{"timeInForce":"newTimeInForce"}`), &obj)
	assert.ErrorContains(t, err, `unsupported order timeInForce: "newTimeInForce"`)
}

func TestOrderTimeInForce_String(t *testing.T) {

	for valStr, val := range timeInForceGetMap() {
		assert.Equal(t, val.String(), valStr, "string "+valStr)
		resolve, err := trading.OrderTimeInForceStrToType(valStr)
		assert.NilError(t, err)
		assert.Equal(t, resolve, val, "from string "+valStr)
	}

	defer func() {
		if r := recover(); r != nil {
		} else {
			t.Fatal("not recoverd")
		}
	}()
	_ = trading.OrderTimeInForce(8).String()
	t.Errorf("The code did not panic")
}

func TestOrderTimeInForce_StrToOrderTypeError(t *testing.T) {
	_, err := trading.OrderTimeInForceStrToType("newTime")
	assert.Error(t, err, `unsupported order timeInForce: newTime`)
}
