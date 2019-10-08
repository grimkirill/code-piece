package trading

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
)

type ClientOrderId [32]byte

func (co ClientOrderId) String() string {
	indexEmpty := bytes.IndexByte(co[:], 0)
	if indexEmpty > 0 {
		return string(co[:indexEmpty])
	}
	return string(co[:])
}

func ClientOrderIdStrToType(val string) (ClientOrderId, error) {
	result := [32]byte{}
	if len(val) > 32 {
		return result, errors.New("too long clientOrderId: " + val)
	}
	copy(result[:], []byte(val))
	return result, nil
}

func ClientOrderIdGenerate() ClientOrderId {
	result := [32]byte{}
	b := make([]byte, 16)
	_, err := rand.Read(b)

	if err != nil {
		panic(errors.New("fail get random for generate trader order id: " + err.Error()))
	}
	hex.Encode(result[:], b)
	return result
}

func ClientOrderIdGenerateFast(id int) ClientOrderId {
	result := [32]byte{}
	copy(result[:], []byte(strconv.Itoa(id)))
	return result
}

func (co ClientOrderId) MarshalJSON() ([]byte, error) {
	indexEmpty := bytes.IndexByte(co[:], 0)
	if indexEmpty == -1 {
		indexEmpty = len(co)
	}
	if indexEmpty == 0 {
		return nil, errors.New("fail marshal empty clientOrderId")
	}
	result := make([]byte, indexEmpty+2)
	result[0] = '"'
	copy(result[1:], co[:])
	result[indexEmpty+1] = '"'

	return result, nil
}

func (co *ClientOrderId) UnmarshalJSON(data []byte) error {
	if len(data) < 3 {
		return errors.New("too small length of clientOrderId: " + string(data))
	}
	if len(data) > 34 {
		return errors.New("too long clientOrderId: " + string(data))
	}
	copy(co[:], data[1:len(data)-1])
	return nil
}
