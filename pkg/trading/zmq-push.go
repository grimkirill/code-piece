package trading

import (
	"sync"
	"time"

	"sync/atomic"

	"github.com/json-iterator/go"
	"github.com/pebbe/zmq4"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var requestCounters = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "zmq_request_count",
	Help: "zmq send message counters",
}, []string{"gate", "type"})

func init() {
	prometheus.MustRegister(requestCounters)
}

type zmqPushConnection struct {
	logger  *zap.Logger
	soc     *zmq4.Socket
	token   string
	login   string
	addr    string
	sendMx  sync.Mutex
	ready   chan bool
	isReady uint32
}

func (c *zmqPushConnection) Ready() chan bool {
	return c.ready
}

func (c *zmqPushConnection) SendRequestNew(order payloadOrderNew) error {
	order.Login = c.login
	data, err := jsoniter.Marshal(transportRequestNew{Data: order, Token: c.token})
	if err != nil {
		return errors.WithMessage(err, "fail marshal request order new:")
	}
	requestCounters.WithLabelValues(c.addr, "new").Inc()

	c.logger.Info("zmq: send", zap.ByteString("msg", data), zap.String("gate", c.addr))

	c.sendMx.Lock()
	defer c.sendMx.Unlock()
	_, err = c.soc.SendBytes(data, zmq4.DONTWAIT)

	if err != nil {
		c.logger.Error("zmq: fail send", zap.ByteString("msg", data), zap.String("gate", c.addr), zap.Error(err))
		return errors.WithMessage(err, "fail send via zmq request order new:")
	}
	return nil
}

func (c *zmqPushConnection) SendRequestCancel(order requestOrderCancel) error {
	order.Login = c.login
	data, err := jsoniter.Marshal(transportRequestCancel{Data: order, Token: c.token})
	if err != nil {
		return errors.WithMessage(err, "fail marshal request order cancel:")
	}
	requestCounters.WithLabelValues(c.addr, "cancel").Inc()

	c.logger.Info("zmq: send", zap.ByteString("msg", data), zap.String("gate", c.addr))

	c.sendMx.Lock()
	defer c.sendMx.Unlock()
	_, err = c.soc.SendBytes(data, zmq4.DONTWAIT)

	if err != nil {
		c.logger.Error("zmq: fail send", zap.ByteString("msg", data), zap.String("gate", c.addr), zap.Error(err))
		return errors.WithMessage(err, "fail send via zmq request order cancel:")
	}
	return nil
}

func (c *zmqPushConnection) SendRequestReplace(order requestOrderReplacePayload) error {
	order.Login = c.login
	data, err := jsoniter.Marshal(transportRequestReplace{Data: order, Token: c.token})
	if err != nil {
		return errors.WithMessage(err, "fail marshal request order replace:")

	}
	requestCounters.WithLabelValues(c.addr, "replace").Inc()

	c.logger.Info("zmq: send", zap.ByteString("msg", data), zap.String("gate", c.addr))

	c.sendMx.Lock()
	defer c.sendMx.Unlock()
	_, err = c.soc.SendBytes(data, zmq4.DONTWAIT)

	if err != nil {
		c.logger.Error("zmq: fail send", zap.ByteString("msg", data), zap.String("gate", c.addr), zap.Error(err))
		return errors.WithMessage(err, "fail send via zmq request order replace:")
	}
	return nil
}

func (c *zmqPushConnection) IsReady() bool {
	return atomic.LoadUint32(&c.isReady) == 1
}

func (c *zmqPushConnection) setReady(val bool) {
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

func (c *zmqPushConnection) Close() error {
	return c.soc.Close()
}

func (c *zmqPushConnection) String() string {
	return "PUSH:" + c.addr
}

func createZmqPushConnection(zmqCtx *zmq4.Context, monitorAddr, addr, publicKey string) (*zmq4.Socket, error) {
	sock, err := zmqCtx.NewSocket(zmq4.PUSH)
	if err != nil {
		return nil, errors.WithMessage(err, "fail create socket")
	}
	if err = sock.Monitor(monitorAddr, zmq4.EVENT_ALL); err != nil {
		return nil, errors.WithMessage(err, "fail set monitor address")
	}

	if err = sock.SetReconnectIvl(time.Second); err != nil {
		return nil, errors.WithMessage(err, "fail set reconnect interval")
	}
	if err = sock.SetSndhwm(100000); err != nil {
		return nil, errors.WithMessage(err, "fail set send buffer messages count")
	}
	if err = sock.SetLinger(time.Duration(5) * time.Second); err != nil {
		return nil, errors.WithMessage(err, "fail set linger timeout")
	}
	if err = sock.SetConnectTimeout(time.Duration(5) * time.Second); err != nil {
		return nil, errors.WithMessage(err, "fail set connect timeout")
	}
	if err = sock.SetHeartbeatIvl(time.Duration(2) * time.Second); err != nil {
		return nil, errors.WithMessage(err, "fail set heartbeat interval")
	}
	if err = sock.SetHeartbeatTimeout(time.Duration(5) * time.Second); err != nil {
		return nil, errors.WithMessage(err, "fail set heartbeat timeout")
	}
	if err = sock.SetImmediate(true); err != nil {
		return nil, errors.WithMessage(err, "fail set immediate send flag")
	}

	if publicKey != "" {
		// auth zmq using curve algorithm
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

// CreatePushConnection create connection, connect, handle status and run send handles
func CreatePushConnection(addr, publicKey string, token string, logger *zap.Logger) (*zmqPushConnection, error) {
	zmqCtx, err := zmq4.NewContext()
	if err != nil {
		return nil, errors.WithMessage(err, "fail create zmq context")
	}
	monitorAddr := generateMonitorAddr()
	online := make(chan bool)
	go runSocketMonitor(zmqCtx, monitorAddr, online, logger)

	sock, err := createZmqPushConnection(zmqCtx, monitorAddr, addr, publicKey)
	if err != nil {
		return nil, errors.WithMessage(err, "fail create push socket")
	}

	push := &zmqPushConnection{
		logger: logger,
		soc:    sock,
		addr:   addr,
		token:  token,
		ready:  make(chan bool, 2),
	}

	if token != "" {
		push.login = "login_42" // Magic!!
	}

	go func() {
		for status := range online {
			push.setReady(status)
		}
	}()

	return push, nil
}
