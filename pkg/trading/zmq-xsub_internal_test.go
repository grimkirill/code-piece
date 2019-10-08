package trading

import (
	"testing"

	"strconv"

	"time"

	"github.com/pebbe/zmq4"
	"go.uber.org/zap"
	"gotest.tools/assert"
)

func TestZmqParceMessagesXubConnection(t *testing.T) {
	timeout := time.After(10 * time.Second)
	done := make(chan bool)
	go func() {
		defer func() {
			done <- true
		}()
		testPushEndpoint := generateListenAddr()
		zmqCtx, err := zmq4.NewContext()
		assert.NilError(t, err)
		defer func() {
			assert.NilError(t, zmqCtx.Term())
		}()
		sock, err := zmqCtx.NewSocket(zmq4.XPUB)
		assert.NilError(t, sock.SetXpubVerbose(1))
		if err != nil {
			t.Error(err)
		}
		assert.NilError(t, sock.Bind(testPushEndpoint))

		log, _ := zap.NewDevelopment()
		connection, err := newXSubConnection(testPushEndpoint, "", "", log)
		assert.NilError(t, err)

		t.Run("send payloads", func(t *testing.T) {

			_, err := sock.RecvBytes(0)
			assert.NilError(t, err)
			_, err = sock.Send(".HEARTBEAT", 0)
			assert.NilError(t, err)
			_, err = sock.RecvBytes(0)
			assert.NilError(t, err)
			_, err = sock.Send(`user_`+strconv.FormatUint(canaryUser, 10)+`.`+"\x00"+`{"Snapshot":{"user":"user_`+strconv.FormatUint(canaryUser, 10)+`","orders":[]}}`+"\x00", 0)
			assert.NilError(t, err)

		})

		t.Run("wait ready", func(t *testing.T) {
			t.Log("wait ready")
			<-connection.Ready()
			assert.NilError(t, connection.Subscribe(101499))
			assert.Check(t, connection.IsReady(), "ready state")
		})

		t.Run("send user snap", func(t *testing.T) {
			_, err = sock.RecvBytes(0)
			assert.NilError(t, err)

			_, err := sock.Send(`user_101499.`+"\x00"+`{"Snapshot":{"user":"user_101499",
                 "orders": [
                   {
                     "orderId": "84230381168",
                     "latestOrderId": "84230381168",
                     "clientOrderId": "a22c24992074fba3cb23b6651c8bebd3",
                     "orderStatus": "new",
                     "participateDoNotInitiate": false,
                     "symbol": "BCNBTC",
                     "side": "sell",
                     "price": "0.1200000000",
                     "quantity": 1,
                     "type": "limit",
                     "timeInForce": "GTC",
                     "lastQuantity": 0,
                     "lastPrice": "",
                     "leavesQuantity": 1,
                     "cumQuantity": 0,
                     "averagePrice": "0",
                     "created": 1547484260602
                   }
             	]
             }}`+"\x00", 0)
			assert.NilError(t, err)
		})

		t.Run("wait reports", func(t *testing.T) {
			for i := 0; i < 2; i++ {
				r := <-connection.Reports()
				log.Info("report", zap.Reflect("r", r))
			}
		})
		assert.NilError(t, sock.Unbind(testPushEndpoint))
		assert.NilError(t, sock.Close())
	}()

	select {
	case <-timeout:
		t.Fatal("zmq XsubParse messages Connection not finish in time")
	case <-done:
	}
}
