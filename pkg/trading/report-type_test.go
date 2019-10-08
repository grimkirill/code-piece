package trading_test

import (
	"encoding/json"
	"testing"

	"github.com/json-iterator/go"
	"gitlab.heather.loc/helios/astra/pkg/trading"
	"gotest.tools/assert"
)

type testReportType struct {
	Report trading.ReportType `json:"report"`
}

func reportTypeGetMap() map[string]trading.ReportType {
	return map[string]trading.ReportType{
		"new":       trading.ReportTypeNew,
		"suspended": trading.ReportTypeSuspended,
		"canceled":  trading.ReportTypeCanceled,
		"rejected":  trading.ReportTypeRejected,
		"expired":   trading.ReportTypeExpired,
		"trade":     trading.ReportTypeTrade,
		"status":    trading.ReportTypeStatus,
		"replaced":  trading.ReportTypeReplaced,
	}
}

func TestReportType_MarshalJSON(t *testing.T) {
	var err error
	var result []byte
	var obj testReportType

	for valStr, val := range reportTypeGetMap() {
		jsonObj := testReportType{Report: val}
		jsonStr := `{"report":"` + valStr + `"}`

		result, err = json.Marshal(&jsonObj)
		assert.NilError(t, err)
		assert.Equal(t, string(result), jsonStr, "std marshal "+valStr)

		result, err = jsoniter.Marshal(&jsonObj)
		assert.NilError(t, err)
		assert.Equal(t, string(result), jsonStr, "jsoniter marshal "+valStr)

		err = json.Unmarshal([]byte(jsonStr), &obj)
		assert.NilError(t, err)
		assert.Equal(t, obj.Report, val, "std unmarshal "+valStr)

		err = jsoniter.Unmarshal([]byte(jsonStr), &obj)
		assert.NilError(t, err)
		assert.Equal(t, obj.Report, val, "jsoniter unmarshal "+valStr)
	}

	_, err = json.Marshal(&testReportType{Report: trading.ReportType(100)})
	assert.ErrorContains(t, err, `invalid report type json conversion: 100`)

	_, err = jsoniter.Marshal(&testReportType{Report: trading.ReportType(100)})
	assert.ErrorContains(t, err, `invalid report type json conversion: 100`)

	err = json.Unmarshal([]byte(`{"report":"newReport"}`), &obj)
	assert.ErrorContains(t, err, `unsupported report type: "newReport"`)

	err = jsoniter.Unmarshal([]byte(`{"report":"newReport"}`), &obj)
	assert.ErrorContains(t, err, `unsupported report type: "newReport"`)
}

func TestReportType_String(t *testing.T) {

	for valStr, val := range reportTypeGetMap() {
		assert.Equal(t, val.String(), valStr, "string "+valStr)
	}

	defer func() {
		if r := recover(); r != nil {
		} else {
			t.Fatal("not recoverd")
		}
	}()
	_ = trading.ReportType(100).String()
	t.Errorf("The code did not panic")
}
