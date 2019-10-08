package trading

type ErrorResponse uint8

const (
	ErrorUnknown ErrorResponse = iota
	ErrorDuplicateClientOrder
	ErrorNotFoundOrder
	ErrorExceedsLimit
	ErrorNotChanged
	ErrorBadQuantity
	ErrorBadPrice
	ErrorTradingNotStarted
	ErrorNoExpireTime
	ErrorTooLateToEnter
)

func (e ErrorResponse) Error() string {
	return errorMapping[e]
}

var errorMapping = map[ErrorResponse]string{
	ErrorUnknown:              "unknown",
	ErrorDuplicateClientOrder: orderRejectDuplicate,
	ErrorNotFoundOrder:        orderRejectNotFound,
	ErrorExceedsLimit:         orderRejectExceedsLimit,
	ErrorNotChanged:           orderIsNotChanged,
	ErrorBadQuantity:          orderRejectBadQuantity,
	ErrorBadPrice:             orderRejectBadPrice,
	ErrorTradingNotStarted:    orderRejectTradingNotStarted,
	ErrorNoExpireTime:         orderRejectNoExpireTime,
	ErrorTooLateToEnter:       orderRejectTooLateToEnter,
}

func getErrorByReject(code string) ErrorResponse {
	if e, ok := errorUnMapping[code]; ok {
		return e
	}
	return 0
}

var errorUnMapping = map[string]ErrorResponse{
	orderRejectDuplicate:         ErrorDuplicateClientOrder,
	orderRejectNotFound:          ErrorNotFoundOrder,
	orderRejectExceedsLimit:      ErrorExceedsLimit,
	orderIsNotChanged:            ErrorNotChanged,
	orderRejectBadQuantity:       ErrorBadQuantity,
	orderRejectBadPrice:          ErrorBadPrice,
	orderRejectTradingNotStarted: ErrorTradingNotStarted,
	orderRejectNoExpireTime:      ErrorNoExpireTime,
	orderRejectTooLateToEnter:    ErrorTooLateToEnter,
}

// nolint: varcheck, deadcode, unused
const (
	orderRejectExceedsLimit              = "orderExceedsLimit"
	orderRejectNotFound                  = "orderNotFound"
	orderRejectDuplicate                 = "duplicateOrder"
	orderRejectBadSide                   = "badSide"
	orderRejectBadType                   = "badType"
	orderRejectBadTimeInForce            = "badTimeInForce"
	orderRejectBadTimeInForceCombination = "badTypeTimeInForceCombination"
	orderRejectBadQuantity               = "badQuantity"
	orderRejectBadPrice                  = "badPrice"
	orderRejectUnknownUser               = "unknownUser"
	orderRejectUnknownSymbol             = "unknownSymbol"
	orderRejectUnknownAccount            = "unknownAccount"
	orderRejectPermissionDeny            = "permissionDenied"
	orderRejectTooLateToEnter            = "tooLateToEnter"
	orderRejectSelfTrade                 = "selfTrade"
	orderRejectExchangeClosed            = "exchangeClosed"
	orderRejectNoExpireTime              = "expireTimeIsNotSpecified"
	orderRejectToLongAccount             = "tooLongAccount"
	orderRejectBadOrderID                = "badOrderId"
	orderRejectTradingNotStarted         = "tradingNotStarted"
	orderRejectSymbolExpired             = "symbolExpired"
	orderRejectAlreadyActivated          = "orderIsAlreadyActivated"
	orderIsNotChanged                    = "orderIsNotChanged"
)
