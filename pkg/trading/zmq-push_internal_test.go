package trading

import (
	"testing"

	"net"

	"fmt"

	"time"

	"github.com/pebbe/zmq4"
	"go.uber.org/zap"
	"gotest.tools/assert"
)

func generateListenAddr() string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("zmqtest: failed to listen on a port: %v", err))
	}
	l.Close()
	return "tcp://" + l.Addr().String()
}

func TestZmqMessagesWithCurvePushConnection(t *testing.T) {
	timeout := time.After(10 * time.Second)
	done := make(chan bool)

	go func() {
		defer func() {
			done <- true
		}()
		testPushEndpoint := generateListenAddr()
		zmqCtx, err := zmq4.NewContext()
		assert.NilError(t, err)

		sock, err := zmqCtx.NewSocket(zmq4.PULL)
		if err != nil {
			t.Error(err)
		}
		assert.NilError(t, sock.SetCurveServer(1))
		assert.NilError(t, sock.SetCurveSecretkey("<LT$k5pT9h@cUf#>}emn5?%jb8smuADKxRcY3]8R"))

		assert.NilError(t, sock.Bind(testPushEndpoint))

		logger, _ := zap.NewDevelopment()
		push, err := CreatePushConnection(testPushEndpoint, "ZBF0hh6dpg}7{DpN&!=AUEwCqSA9C!G*@q7Rr&j.", "", logger)
		if err != nil {
			t.Error(err)
		}
		defer push.Close()

		t.Run("wait push ready", func(t *testing.T) {
			t.Log("wait ready...")
			<-push.Ready()
			t.Log("wait push done")
			time.Sleep(time.Second)
		})

		t.Run("send requests", func(t *testing.T) {
			err = push.SendRequestNew(payloadOrderNew{
				Quantity:      "1",
				ClientOrderID: ClientOrderIdGenerateFast(10123),
				TimeInForce:   OrderTimeInForceGTC,
				Type:          OrderTypeMarket,
				Symbol:        "EURUSD",
				Side:          OrderSideSell,
				PostOnly:      false,
				UserID:        messageUserID(122001),
			})
			assert.NilError(t, err)
			price := "1.0146"
			err = push.SendRequestNew(payloadOrderNew{
				Quantity:      "1",
				ClientOrderID: ClientOrderIdGenerateFast(10124),
				TimeInForce:   OrderTimeInForceGTC,
				Type:          OrderTypeStopLimit,
				Symbol:        "EURUSD",
				Price:         price,
				StopPrice:     price,
				Side:          OrderSideSell,
				PostOnly:      false,
				UserID:        messageUserID(122001),
			})
			assert.NilError(t, err)
			err = push.SendRequestCancel(requestOrderCancel{
				ClientOrderID:              ClientOrderIdGenerateFast(10123),
				Symbol:                     "EURUSD",
				Side:                       OrderSideSell,
				UserID:                     messageUserID(122001),
				CancelRequestClientOrderID: ClientOrderIdGenerateFast(20123),
			})
			assert.NilError(t, err)
			err = push.SendRequestReplace(requestOrderReplacePayload{
				ClientOrderID:              ClientOrderIdGenerateFast(10124),
				Side:                       OrderSideSell,
				Price:                      "1.0145",
				Quantity:                   "2",
				Symbol:                     "EURUSD",
				Type:                       OrderTypeLimit,
				TimeInForce:                OrderTimeInForceGTC,
				UserID:                     messageUserID(122001),
				CancelRequestClientOrderID: ClientOrderIdGenerateFast(10125),
			})
			assert.NilError(t, err)
			t.Log("send requests done")
		})

		t.Run("wait data in socket", func(t *testing.T) {
			t.Log("wait new order 1")
			recvNewMessage1, err := sock.RecvBytes(0)
			assert.NilError(t, err)
			expectedNewMessage1 := `{"NewOrder":{"userId":"user_122001","clientOrderId":"10123","type":"market","side":"sell","timeInForce":"GTC","participateDoNotInitiate":false,"symbol":"EURUSD","quantity":"1"}}`
			assert.Equal(t, string(recvNewMessage1), expectedNewMessage1, "new order 1")

			t.Log("wait new order 2")
			recvNewMessage2, err := sock.RecvBytes(0)
			assert.NilError(t, err)
			expectedNewMessage2 := `{"NewOrder":{"userId":"user_122001","clientOrderId":"10124","type":"stopLimit","side":"sell","timeInForce":"GTC","participateDoNotInitiate":false,"symbol":"EURUSD","quantity":"1","price":"1.0146","stopPrice":"1.0146"}}`
			assert.Equal(t, string(recvNewMessage2), expectedNewMessage2, "new order 2")

			t.Log("wait order cancel")
			recvCancelMessage, err := sock.RecvBytes(0)
			assert.NilError(t, err)
			expectedCancelMessage := `{"OrderCancel":{"userId":"user_122001","clientOrderId":"10123","cancelRequestClientOrderId":"20123","symbol":"EURUSD","side":"sell"}}`
			assert.Equal(t, string(recvCancelMessage), expectedCancelMessage, "cancel order")

			t.Log("wait order replace")
			recvReplaceMessage, err := sock.RecvBytes(0)
			assert.NilError(t, err)
			expectedReplaceMessage := `{"CancelReplaceOrder":{"userId":"user_122001","clientOrderId":"10124","cancelRequestClientOrderId":"10125","symbol":"EURUSD","side":"sell","type":"limit","timeInForce":"GTC","quantity":"2","price":"1.0145"}}`
			assert.Equal(t, string(recvReplaceMessage), expectedReplaceMessage, "replace order")
		})
		t.Log("test complete")
	}()

	select {
	case <-timeout:
		t.Fatal("zmq WithCurvePushConnection not finish in time")
	case <-done:
	}
}
