package trading

import "time"

// special reserved user, with zero balance
const canaryUser = 100001
const canaryTestDuration = time.Second * 5

var (
	canaryTestSymbol   = "BCNBTC"
	canaryTestPrice    = "1"
	canaryTestQuantity = "1"
)
