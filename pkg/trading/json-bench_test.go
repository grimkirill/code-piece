package trading

import (
	"testing"

	"github.com/json-iterator/go"
)

func BenchmarkMessageReport_JsonUnmarshal(b *testing.B) {
	reportBytes := []byte(`{"ExecutionReport":{"orderId":"84230381170","latestOrderId":"84230381170","clientOrderId":"db79c4da6a18334bfbbdee47812e48d0","orderStatus":"new","participateDoNotInitiate":false,"userId":"user_101499","symbol":"BCNBTC","side":"sell","price":"0.1200000000","quantity":1,"type":"limit","timeInForce":"GTC","lastQuantity":0,"lastPrice":"","leavesQuantity":1,"cumQuantity":0,"averagePrice":"0","created":1547549876902,"execReportType":"new","timestamp":1547549876902}}`)

	for i := 0; i < b.N; i++ {
		var rep messageReport
		err := jsoniter.Unmarshal(reportBytes, &rep)
		if err != nil {
			b.Fatal("fail parse report", err)
		}
	}
}
