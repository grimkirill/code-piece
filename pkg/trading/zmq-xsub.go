package trading

import (
	"bytes"
	"strconv"
	"sync"
	"time"

	"sync/atomic"

	jsoniter "github.com/json-iterator/go"
	"github.com/pebbe/zmq4"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var messageCounters = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "zmq_message_count",
	Help: "zmq income message counters",
}, []string{"gate", "type"})

var rejectCounters = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "zmq_reject_count",
	Help: "zmq income reject counters",
}, []string{"gate", "type"})

func init() {
	prometheus.MustRegister(messageCounters, rejectCounters)
}

// zmqXsubConnection provide subscription for zmq_transactional_gate
type zmqXsubConnection struct {
	soc     *zmq4.Socket
	addr    string
	token   string
	reports chan interface{}
	mx      sync.Mutex
	logger  *zap.Logger
	ready   chan bool
	isReady uint32
}

func (c *zmqXsubConnection) Reports() chan interface{} {
	return c.reports
}

func (c *zmqXsubConnection) Ready() chan bool {
	return c.ready
}

// GetAddr get connection endpoint address
func (c *zmqXsubConnection) GetAddr() string {
	return c.addr
}

// IsReady return connection ready state.
// If connection established and heartbeat is received, then true
func (c *zmqXsubConnection) IsReady() bool {
	return atomic.LoadUint32(&c.isReady) == 1
}

func (c *zmqXsubConnection) Close() error {
	return c.soc.Close()
}

func newXSubSocket(zmqCtx *zmq4.Context, monitorAddr, addr, publicKey string) (*zmq4.Socket, error) {
	sock, err := zmqCtx.NewSocket(zmq4.XSUB)
	if err != nil {
		return nil, errors.WithMessage(err, "fail create socket")
	}

	if err = sock.Monitor(monitorAddr, zmq4.EVENT_ALL); err != nil {
		return nil, errors.WithMessage(err, "fail set monitor address")
	}

	if err = sock.SetReconnectIvl(time.Second); err != nil {
		return nil, errors.WithMessage(err, "fail set reconnect interval")
	}
	if err = sock.SetConnectTimeout(time.Duration(5) * time.Second); err != nil {
		return nil, errors.WithMessage(err, "fail set connect timeout")
	}
	if err = sock.SetHeartbeatIvl(time.Duration(10) * time.Second); err != nil {
		return nil, errors.WithMessage(err, "fail set heartbeat interval")
	}
	if err = sock.SetHeartbeatTimeout(time.Duration(20) * time.Second); err != nil {
		return nil, errors.WithMessage(err, "fail set heartbeat timeout")
	}

	if err = sock.SetSndhwm(100000); err != nil {
		return nil, errors.WithMessage(err, "fail set send buffer messages count")
	}

	if publicKey != "" {
		var keyPublic, keySecret string
		keyPublic, keySecret, err = zmq4.NewCurveKeypair()
		if err != nil {
			return nil, errors.WithMessage(err, "fail generate curve pair")

		}
		if err = sock.ClientAuthCurve(publicKey, keyPublic, keySecret); err != nil {
			return nil, errors.WithMessage(err, "fail set auth curve")
		}
	}

	if err = sock.Connect(addr); err != nil {
		return nil, errors.WithMessage(err, "fail connect "+addr)
	}

	return sock, nil
}

// newXSubConnection create zmq socket with monitoring
// init subscribe and receive handler
func newXSubConnection(addr, publicKey string, token string, logger *zap.Logger) (xsubConnecter, error) {
	zmqCtx, err := zmq4.NewContext()
	if err != nil {
		return nil, errors.WithMessage(err, "fail create zmq context")
	}
	monitorAddr := generateMonitorAddr()
	online := make(chan bool)
	go runSocketMonitor(zmqCtx, monitorAddr, online, logger)

	sock, err := newXSubSocket(zmqCtx, monitorAddr, addr, publicKey)
	if err != nil {
		return nil, errors.WithMessage(err, "fail create socket")
	}

	_, err = sock.SendBytes([]byte("\x01.HEARTBEAT"), zmq4.DONTWAIT)
	if err != nil {
		return nil, err
	}

	xsub := &zmqXsubConnection{
		soc:     sock,
		addr:    addr,
		logger:  logger,
		ready:   make(chan bool, 2),
		reports: make(chan interface{}, 1000),
		token:   token,
	}

	err = xsub.Subscribe(canaryUser)
	if err != nil {
		return nil, err
	}

	go xsub.handleMonitor(online)

	go xsub.readMessages()

	return xsub, nil
}

func (c *zmqXsubConnection) setReady(val bool) {
	var state uint32
	if val {
		state = 1
	}

	if atomic.SwapUint32(&c.isReady, state) != state {
		if val {
			c.logger.Info("zmq: connection ready", zap.String("addr", c.addr))
		} else {
			c.logger.Warn("zmq: connection closed", zap.String("addr", c.addr))
		}
		select {
		case c.ready <- val:
			// ok
		default:
			panic("zmq: discarding ready call chan capacity")
		}
	}
}

func (c *zmqXsubConnection) handleMonitor(online chan bool) {
	for status := range online {
		if !status && c.IsReady() {
			c.logger.Info("zmq: connection closed", zap.String("addr", c.addr))
		}
		if !status {
			c.setReady(false)
		}
	}
}

func (c *zmqXsubConnection) doHeartbeat() {
	c.setReady(true)
}

// Subscribe allow subscribe for reports for user.
func (c *zmqXsubConnection) Subscribe(userID uint64) error {
	c.logger.Info("zmq: subscribe", zap.String("addr", c.addr), zap.Uint64("UserID", userID))
	c.mx.Lock()
	defer c.mx.Unlock()
	_, err := c.soc.SendBytes([]byte("\x01user_"+strconv.FormatUint(userID, 10)+"."+c.token), zmq4.DONTWAIT)
	if err != nil {
		c.logger.Error("zmq: fail send subscription payload", zap.Error(err))
	}
	return err
}

// UnSubscribe allow unsubscribe for reports for user
func (c *zmqXsubConnection) UnSubscribe(userID uint64) error {
	c.logger.Info("zmq: unsubscribe", zap.String("addr", c.addr), zap.Uint64("UserID", userID))
	c.mx.Lock()
	defer c.mx.Unlock()
	_, err := c.soc.SendBytes([]byte("\x00user_"+strconv.FormatUint(userID, 10)+"."+c.token), zmq4.DONTWAIT)
	if err != nil {
		c.logger.Error("zmq: fail send unsubscription payload", zap.Error(err))
	}
	return err
}

func (c *zmqXsubConnection) readMessages() {
	dotBytes := []byte(".")[0]

	heartbeat := []byte(".HEARTBEAT")
	snapshot := []byte("{\"Snapshot\":")
	report := []byte("{\"ExecutionReport\":")
	reject := []byte("{\"CancelReject\":")
	rejectReplace := []byte("{\"CancelReplaceReject\":")
	subscription := []byte("{\"SubscriptionReply\":")

	for {

		msg, err := c.soc.RecvBytes(0)
		if err != nil {
			c.logger.Fatal("zmq: receive data error", zap.Error(err), zap.String("addr", c.addr))
			panic(err)
		}
		c.doHeartbeat()
		//c.logger.Info("zmq: income", zap.String("payload", string(msg)))
		if bytes.Contains(msg, heartbeat) {
			messageCounters.WithLabelValues(c.addr, "heartbeat").Inc()
			c.logger.Info("zmq: heartbeat", zap.String("addr", c.addr))
		} else {
			dotIndex := bytes.IndexByte(msg, dotBytes) + len(c.token)
			if bytes.Contains(msg, snapshot) {
				messageCounters.WithLabelValues(c.addr, "snapshot").Inc()
				var snap messageSnapshot
				err := jsoniter.Unmarshal(msg[dotIndex+2:len(msg)-1], &snap)
				if err != nil {
					c.logger.Error("zmq: parse fail snapshot", zap.Error(err), zap.String("msg", string(msg[dotIndex+2:len(msg)-1])))
				} else {
					c.logger.Info("zmq: snapshot ", zap.String("addr", c.addr), zap.String("user", snap.Data.User), zap.Int("orders count", len(snap.Data.Orders)))
					userID := userStrToID(snap.Data.User)
					orders := make([]OrderModel, 0, len(snap.Data.Orders))
					for _, orderMessage := range snap.Data.Orders {
						orders = append(orders, orderMessage.CreateOrderModel())
					}
					c.reports <- &payloadSnapshot{userID: userID, orders: orders}
				}
			} else if bytes.Contains(msg, report) {
				var report messageReport
				err := jsoniter.Unmarshal(msg[dotIndex+2:len(msg)-1], &report)
				if err != nil {
					c.logger.Error("zmq: parse fail report", zap.Error(err), zap.String("msg", string(msg[dotIndex+2:len(msg)-1])))
				} else {
					messageCounters.WithLabelValues(c.addr, report.Data.ExecReportType.String()).Inc()
					userID := userStrToID(report.Data.UserID)
					if report.Data.ExecReportType == ReportTypeRejected {
						rejectCounters.WithLabelValues(c.addr, report.Data.OrderRejectReason).Inc()
						rejectData := &payloadReject{
							userID:        userID,
							ClientOrderID: report.Data.ClientOrderID,
							Timestamp:     report.Data.Timestamp,
							RejectReason:  report.Data.OrderRejectReason,
						}
						c.reports <- rejectData
					} else {
						c.reports <- &payloadReport{userID: userID, report: report.Data.CreateOrderReport()}
					}
				}
			} else if bytes.Contains(msg, reject) {
				messageCounters.WithLabelValues(c.addr, "rejectCancel").Inc()
				var reject messageCancelContent
				err := jsoniter.Unmarshal(msg[dotIndex+2:len(msg)-1], &reject)
				if err != nil {
					c.logger.Error("zmq: parse fail reject cancel", zap.Error(err), zap.String("msg", string(msg[dotIndex+2:len(msg)-1])))
				} else {
					rejectCounters.WithLabelValues(c.addr, reject.CancelReject.RejectReasonCode).Inc()

					if reject.CancelReject.UserID == "" {
						reject.CancelReject.UserID = string(msg[:dotIndex-len(c.token)])
					}

					userID := userStrToID(reject.CancelReject.UserID)
					rejectData := &payloadReject{
						userID:                     userID,
						ClientOrderID:              reject.CancelReject.ClientOrderID,
						CancelRequestClientOrderID: reject.CancelReject.CancelRequestClientOrderID,
						RejectReason:               reject.CancelReject.RejectReasonCode,
					}
					c.reports <- rejectData
				}
			} else if bytes.Contains(msg, rejectReplace) {
				messageCounters.WithLabelValues(c.addr, "rejectReplace").Inc()
				var reject messageCancelReplaceContent
				err := jsoniter.Unmarshal(msg[dotIndex+2:len(msg)-1], &reject)
				if err != nil {
					c.logger.Error("zmq: parse fail reject replace", zap.Error(err), zap.String("msg", string(msg[dotIndex+2:len(msg)-1])))
				} else {
					rejectCounters.WithLabelValues(c.addr, reject.CancelReject.RejectReasonCode).Inc()

					if reject.CancelReject.UserID == "" {
						reject.CancelReject.UserID = string(msg[:dotIndex-len(c.token)])
					}

					userID := userStrToID(reject.CancelReject.UserID)
					rejectData := &payloadReject{
						userID:                     userID,
						ClientOrderID:              reject.CancelReject.ClientOrderID,
						CancelRequestClientOrderID: reject.CancelReject.CancelRequestClientOrderID,
						RejectReason:               reject.CancelReject.RejectReasonCode,
					}
					c.reports <- rejectData
				}
			} else if bytes.Contains(msg, subscription) {
				var reply subscribeReplyContent
				err := jsoniter.Unmarshal(msg[dotIndex+2:len(msg)-1], &reply)
				if err != nil {
					c.logger.Error("zmq: parse fail subscription", zap.Error(err), zap.String("msg", string(msg[dotIndex+2:len(msg)-1])))
				} else {
					c.logger.Info("zmq: subscription income < " + string(msg))
				}
			} else {
				c.logger.Error("zmq: income < " + string(msg))
			}
		}
	}
}

// nolint: unused
// subscribeAll allow subscribe for all reports (for debug zmq purpose)
func (c *zmqXsubConnection) subscribeAll() error {
	c.mx.Lock()
	defer c.mx.Unlock()
	data := []byte("\x01")
	_, err := c.soc.SendBytes(data, zmq4.DONTWAIT)
	return err
}
