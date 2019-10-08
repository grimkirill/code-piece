package trading

import (
	"bytes"
	"errors"
	"strconv"
)

type ReportType uint8

const (
	ReportTypeNew ReportType = iota
	ReportTypeSuspended
	ReportTypeCanceled
	ReportTypeRejected
	ReportTypeExpired
	ReportTypeTrade
	ReportTypeStatus
	ReportTypeReplaced

	reportTypeNewStr       = "new"
	reportTypeSuspendedStr = "suspended"
	reportTypeCanceledStr  = "canceled"
	reportTypeRejectedStr  = "rejected"
	reportTypeExpiredStr   = "expired"
	reportTypeTradeStr     = "trade"
	reportTypeStatusStr    = "status"
	reportTypeReplacedStr  = "replaced"
)

var (
	reportTypeNewByte       = []byte(`"new"`)
	reportTypeSuspendedByte = []byte(`"suspended"`)
	reportTypeCanceledByte  = []byte(`"canceled"`)
	reportTypeRejectedByte  = []byte(`"rejected"`)
	reportTypeExpiredByte   = []byte(`"expired"`)
	reportTypeTradeByte     = []byte(`"trade"`)
	reportTypeStatusByte    = []byte(`"status"`)
	reportTypeReplacedByte  = []byte(`"replaced"`)
)

func (rt ReportType) String() string {
	switch rt {
	case ReportTypeNew:
		return reportTypeNewStr
	case ReportTypeSuspended:
		return reportTypeSuspendedStr
	case ReportTypeCanceled:
		return reportTypeCanceledStr
	case ReportTypeRejected:
		return reportTypeRejectedStr
	case ReportTypeExpired:
		return reportTypeExpiredStr
	case ReportTypeTrade:
		return reportTypeTradeStr
	case ReportTypeStatus:
		return reportTypeStatusStr
	case ReportTypeReplaced:
		return reportTypeReplacedStr

	}
	panic("invalid report type string conversion" + strconv.Itoa(int(rt)))
}

func (rt ReportType) MarshalJSON() ([]byte, error) {
	switch rt {
	case ReportTypeNew:
		return reportTypeNewByte, nil

	case ReportTypeSuspended:
		return reportTypeSuspendedByte, nil

	case ReportTypeCanceled:
		return reportTypeCanceledByte, nil

	case ReportTypeRejected:
		return reportTypeRejectedByte, nil

	case ReportTypeExpired:
		return reportTypeExpiredByte, nil

	case ReportTypeTrade:
		return reportTypeTradeByte, nil

	case ReportTypeStatus:
		return reportTypeStatusByte, nil

	case ReportTypeReplaced:
		return reportTypeReplacedByte, nil

	}
	return nil, errors.New("invalid report type json conversion: " + strconv.Itoa(int(rt)))
}

func (rt *ReportType) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, reportTypeNewByte) {
		*rt = ReportTypeNew
		return nil
	}
	if bytes.Equal(data, reportTypeCanceledByte) {
		*rt = ReportTypeCanceled
		return nil
	}
	if bytes.Equal(data, reportTypeReplacedByte) {
		*rt = ReportTypeReplaced
		return nil
	}
	if bytes.Equal(data, reportTypeRejectedByte) {
		*rt = ReportTypeRejected
		return nil
	}
	if bytes.Equal(data, reportTypeTradeByte) {
		*rt = ReportTypeTrade
		return nil
	}
	if bytes.Equal(data, reportTypeExpiredByte) {
		*rt = ReportTypeExpired
		return nil
	}
	if bytes.Equal(data, reportTypeSuspendedByte) {
		*rt = ReportTypeSuspended
		return nil
	}
	if bytes.Equal(data, reportTypeStatusByte) {
		*rt = ReportTypeStatus
		return nil
	}

	return errors.New("unsupported report type: " + string(data))
}
