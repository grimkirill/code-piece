package trading

import (
	"fmt"
	"sync/atomic"

	"github.com/pebbe/zmq4"
	"go.uber.org/zap"
)

// runSocketMonitor monitors zmq socket connection status
func runSocketMonitor(zmqCtx *zmq4.Context, addr string, online chan bool, logger *zap.Logger) {
	s, err := zmqCtx.NewSocket(zmq4.PAIR)
	if err != nil {
		logger.Fatal("zmq-monitor: fail create new socket", zap.Error(err))
		panic(err)
	}
	if err = s.SetLinger(0); err != nil {
		logger.Error("zmq-monitor: fail setLinger", zap.Error(err))
	}
	defer func() {
		if err = s.Close(); err != nil {
			logger.Error("zmq-monitor: fail close socket", zap.Error(err))
		}
	}()

	err = s.Connect(addr)
	if err != nil {
		logger.Fatal("zmq-monitor: fail connect", zap.Error(err))
		panic(err)
	}

	for {
		event, address, _, err := s.RecvEvent(0)
		if err != nil {
			logger.Error("zmq-monitor: fail close socket", zap.Error(err))
		} else {
			if event == zmq4.EVENT_CONNECTED {
				logger.Info("zmq-monitor: connection established", zap.String("addr", address))
				online <- true
			} else if event == zmq4.EVENT_CONNECT_DELAYED {
				logger.Warn("zmq-monitor: trying to connect", zap.String("addr", address))
			} else if event == zmq4.EVENT_CONNECT_RETRIED {
				logger.Warn("zmq-monitor: retry connect", zap.String("addr", address))
			} else if event == zmq4.EVENT_CLOSED {
				logger.Warn("zmq-monitor: closed", zap.String("addr", address))
				online <- false
			} else if event.String() == "<NONE>" {
				logger.Warn("zmq-monitor: stop monitor")
			} else {
				logger.Warn("zmq-monitor: unprocessed event", zap.String("addr", address), zap.String("event", event.String()))
			}
		}
	}
}

var monitorID int64

func generateMonitorAddr() string {
	nextID := atomic.AddInt64(&monitorID, 1)
	return fmt.Sprintf("inproc://monitor_t.%d", nextID)
}
