package trading

import (
	"strings"

	"net/url"

	"strconv"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type configZmqTrader struct {
	Token    string
	XSubAddr string
	XSubKey  string
	PushAddr string
	PushKey  string
}

func parseDsnZmq(dsn string) ([]configZmqTrader, error) {

	configs := strings.Split(strings.Trim(dsn, " "), " ")

	var token, xsubKey, pushKey string
	result := make([]configZmqTrader, 0)

	for _, conf := range configs {
		if strings.HasPrefix(conf, "token=") {
			token = strings.TrimPrefix(conf, "token=")
		}

		if strings.HasPrefix(conf, "xsub_key=") {
			xsubKey = strings.TrimPrefix(conf, "xsub_key=")
		}

		if strings.HasPrefix(conf, "push_key=") {
			pushKey = strings.TrimPrefix(conf, "push_key=")
		}
	}

	for _, conf := range configs {
		if strings.HasPrefix(conf, "zmq://") {
			u, err := url.Parse(conf)
			if err != nil {
				return nil, err
			}
			if u.Hostname() == "" {
				return nil, errors.New("host is empty")
			}

			if u.Port() == "" {
				return nil, errors.New("port is empty")
			}

			pushPort, err := strconv.Atoi(u.Port())
			if err != nil {
				return nil, errors.WithMessage(err, "invalid push port value")
			}
			xsubPort := pushPort + 1

			if u.Query().Get("xsub_port") != "" {
				xsubPort, err = strconv.Atoi(u.Query().Get("xsub_port"))
				if err != nil {
					return nil, errors.WithMessage(err, "invalid xsub port value")
				}
			}

			result = append(result, configZmqTrader{
				PushAddr: "tcp://" + u.Hostname() + ":" + strconv.Itoa(pushPort),
				XSubAddr: "tcp://" + u.Hostname() + ":" + strconv.Itoa(xsubPort),
				Token:    token,
				PushKey:  pushKey,
				XSubKey:  xsubKey,
			})
		}
	}

	if len(result) == 0 {
		return nil, errors.New("empty config")
	}

	return result, nil
}

func createZmqTrader(logger *zap.Logger, cfg configZmqTrader) (Trader, error) {
	xSubConnection, err := newXSubConnection(cfg.XSubAddr, cfg.XSubKey, cfg.Token, logger)
	if err != nil {
		return nil, errors.WithMessage(err, "fail create xsub zmq connection")
	}
	pushConnection, err := CreatePushConnection(cfg.PushAddr, cfg.PushKey, cfg.Token, logger)
	if err != nil {
		return nil, errors.WithMessage(err, "fail create push zmq connection")
	}
	return StartNewClient(logger, pushConnection, xSubConnection), nil
}

type configMockTrader struct {
	Host     string
	Port     int
	Ready    bool
	Fixtures bool
}

func parseDsnMock(dsn string) (*configMockTrader, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	cfg := &configMockTrader{}

	cfg.Host = u.Hostname()

	if u.Port() != "" {
		cfg.Port, err = strconv.Atoi(u.Port())
		if err != nil {
			return nil, errors.WithMessage(err, "invalid push port value")
		}
	}

	if u.Query().Get("ready") == "true" {
		cfg.Ready = true
	}
	if u.Query().Get("fixtures") == "true" {
		cfg.Fixtures = true
	}

	return cfg, nil
}

func NewTrader(logger *zap.Logger, dsn string) (Trader, error) {

	if strings.HasPrefix(dsn, "mock://") {
		cfg, err := parseDsnMock(dsn)
		if err != nil {
			return nil, errors.WithMessage(err, "fail parse mock dsn")
		}
		trader := NewMockTrader(logger)
		trader.SetReady(cfg.Ready)
		if cfg.Fixtures {
			trader.SetupFixtures()
		}
		return trader, nil
	}

	if strings.HasPrefix(dsn, "zmq://") {
		cfg, err := parseDsnZmq(dsn)
		if err != nil {
			return nil, errors.WithMessage(err, "fail parse zmq dsn")
		}
		if len(cfg) == 1 {
			return createZmqTrader(logger, cfg[0])
		}
		gates := make([]Trader, 0, len(cfg))
		for _, gateConf := range cfg {
			gate, err := createZmqTrader(logger, gateConf)
			if err != nil {
				return nil, errors.WithMessage(err, "fail create gate")
			}
			gates = append(gates, gate)
		}
		return newClusterTrader(logger, gates), nil
	}

	return nil, errors.New("config not supported")
}
