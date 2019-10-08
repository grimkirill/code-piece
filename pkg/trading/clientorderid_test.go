package trading_test

import (
	"testing"

	"encoding/json"

	"gitlab.heather.loc/helios/astra/pkg/trading"
	"gotest.tools/assert"
)

type testClientOrderIdContent struct {
	ID trading.ClientOrderId `json:"id"`
}

func TestClientOrderId_UnmarshalJSON(t *testing.T) {
	casesOk := map[string]string{
		"123":                              `{"id":"123"}`,
		"a51c60c7c0ee6728b708aac085f94f72": `{"id":"a51c60c7c0ee6728b708aac085f94f72"}`,
	}

	for val, key := range casesOk {
		var obj testClientOrderIdContent
		err := json.Unmarshal([]byte(key), &obj)
		assert.NilError(t, err)
		assert.Equal(t, obj.ID.String(), val, "json unmarshal "+val)
	}

	t.Run("test small", func(t *testing.T) {
		var obj testClientOrderIdContent
		err := json.Unmarshal([]byte(`{"id":""}`), &obj)
		assert.ErrorContains(t, err, `too small length of clientOrderId`)
	})

	t.Run("test long", func(t *testing.T) {
		var obj testClientOrderIdContent
		err := json.Unmarshal([]byte(`{"id":"a51c60c7c0ee6728b708aac085f94f72_"}`), &obj)
		assert.ErrorContains(t, err, `too long clientOrderId`)
	})
}
