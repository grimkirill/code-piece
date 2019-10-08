package trading

import (
	"testing"

	"gotest.tools/assert"
)

func TestParseDsnZmq(t *testing.T) {

	t.Run("simple ok", func(t *testing.T) {
		cfg, err := parseDsnZmq("zmq://10.195.46.21:7778")
		assert.NilError(t, err)
		assert.Equal(t, cfg[0].PushAddr, "tcp://10.195.46.21:7778")
		assert.Equal(t, cfg[0].XSubAddr, "tcp://10.195.46.21:7779")
	})

	t.Run("simple ok port", func(t *testing.T) {
		cfg, err := parseDsnZmq("zmq://10.195.46.21:7778?xsub_port=8779")
		assert.NilError(t, err)
		assert.Equal(t, cfg[0].PushAddr, "tcp://10.195.46.21:7778")
		assert.Equal(t, cfg[0].XSubAddr, "tcp://10.195.46.21:8779")
	})

	t.Run("complete ok", func(t *testing.T) {
		cfg, err := parseDsnZmq("zmq://10.195.46.21:7778?xsub_port=8779 token=astra   xsub_key=a!BHF!Tb?rtY.gHjwzx!>>V@DJ2ZAlLP6Mp:llwJ push_key=RuU0F(D9u/57m:YLghIvusNgd)r&lfg6.ZJ3SY@A")
		assert.NilError(t, err)
		assert.Equal(t, cfg[0].PushAddr, "tcp://10.195.46.21:7778")
		assert.Equal(t, cfg[0].XSubAddr, "tcp://10.195.46.21:8779")
		assert.Equal(t, cfg[0].Token, "astra")
		assert.Equal(t, cfg[0].PushKey, "RuU0F(D9u/57m:YLghIvusNgd)r&lfg6.ZJ3SY@A")
		assert.Equal(t, cfg[0].XSubKey, "a!BHF!Tb?rtY.gHjwzx!>>V@DJ2ZAlLP6Mp:llwJ")
	})

	t.Run("complete multiply ok", func(t *testing.T) {
		cfg, err := parseDsnZmq("zmq://10.195.46.21:7778 zmq://10.195.46.71:8778?xsub_port=8779 xsub_key=qwe push_key=rty")
		assert.NilError(t, err)
		assert.Equal(t, cfg[0].PushAddr, "tcp://10.195.46.21:7778")
		assert.Equal(t, cfg[0].XSubAddr, "tcp://10.195.46.21:7779")
		assert.Equal(t, cfg[1].PushAddr, "tcp://10.195.46.71:8778")
		assert.Equal(t, cfg[1].XSubAddr, "tcp://10.195.46.71:8779")
		assert.Equal(t, cfg[0].PushKey, "rty")
		assert.Equal(t, cfg[1].PushKey, "rty")
		assert.Equal(t, cfg[0].XSubKey, "qwe")
		assert.Equal(t, cfg[1].XSubKey, "qwe")
	})

}
