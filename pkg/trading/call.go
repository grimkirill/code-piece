package trading

import (
	"log"
	"sync/atomic"
	"time"
)

var nextRequestID uint64

type requestCall struct {
	id                         uint64
	gate                       string
	start                      time.Time
	CancelRequestClientOrderID ClientOrderId
	Reply                      *OrderReport // The reply from the function (*struct).
	Error                      error        // After completion, the error status.
	Done                       chan *requestCall
	trades                     []Trade
	symbol                     string
	side                       OrderSide
	waitComplete               bool
}

func (call *requestCall) done() {

	requestDurations.WithLabelValues(call.gate, "request").Observe(float64(time.Since(call.start) / time.Microsecond))
	select {
	case call.Done <- call:
		// ok
	default:
		// We don't want to block here. It is the caller's responsibility to make
		// sure the channel has enough buffer space. See comment in Go().
		log.Println("zmq: discarding requestCall reply due to insufficient Done chan capacity")

	}
}

func createCall(gate string) *requestCall {
	return &requestCall{
		id:    atomic.AddUint64(&nextRequestID, 1),
		gate:  gate,
		start: time.Now(),
		Done:  make(chan *requestCall, 1),
	}
}

type callSnapshot struct {
	id    uint64
	gate  string
	start time.Time
	Error error // After completion, the error status.
	Done  chan *callSnapshot
}

func (call *callSnapshot) done() {
	requestDurations.WithLabelValues(call.gate, "snapshot").Observe(float64(time.Since(call.start) / time.Microsecond))
	select {
	case call.Done <- call:
		// ok
	default:
		// We don't want to block here. It is the caller's responsibility to make
		// sure the channel has enough buffer space. See comment in Go().
		log.Println("zmq-call: snapshot discarding requestCall reply due to insufficient Done chan capacity")
	}
}

func createCallSnapshot(gate string) *callSnapshot {
	return &callSnapshot{
		gate:  gate,
		id:    atomic.AddUint64(&nextRequestID, 1),
		start: time.Now(),
		Done:  make(chan *callSnapshot, 1),
	}
}
