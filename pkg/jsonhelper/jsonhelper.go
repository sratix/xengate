package jsonhelper

import (
	jsoniter "github.com/json-iterator/go"

	"go.uber.org/zap"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func Encode[T any](t T) []byte {
	b, err := json.Marshal(t)
	if err != nil {
		zap.S().With("t", t).Fatalln("couldn't encode the variable", "error", err)
	}
	return b
}

func Decode[T any](b []byte) T {
	var t T
	err := json.Unmarshal(b, &t)
	if err != nil {
		zap.S().With("t", t).With("val", string(b)).Fatalln("couldn't decode the variable", "error", err)
	}
	return t
}
