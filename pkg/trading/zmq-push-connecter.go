package trading

// pushConnecter carries provide send new trading execution transport
type pushConnecter interface {

	// SendRequestNew send new order request over transport
	SendRequestNew(order payloadOrderNew) error

	// SendRequestCancel send cancel order request over transport
	SendRequestCancel(order requestOrderCancel) error

	// SendRequestReplace send order cancel-replace request over transport
	SendRequestReplace(order requestOrderReplacePayload) error

	// IsReady inform about ready transport status
	IsReady() bool

	// ready  ready transport status
	Ready() chan bool

	Close() error
}
