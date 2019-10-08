package trading

// xsubConnecter carries provide receive trading execution transport
type xsubConnecter interface {

	// IsReady inform about ready transport status
	IsReady() bool

	// Ready  ready transport status
	Ready() chan bool

	// Subscribe for get user snapshot and reports
	Subscribe(userID uint64) error

	UnSubscribe(userID uint64) error

	Reports() chan interface{}

	GetAddr() string

	Close() error
}
