package cyberdb

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
)

func TestDec128ToFloat64(t *testing.T) {
	v, _ := primitive.ParseDecimal128("123456")
	res := Dec128ToFloat64(v, 100)
	if res != 1234.56 {
		t.Errorf("Wrong value from Dec128ToFloat64: wait %f, get %f\n", 1234.56, res)
	}
	res = Dec128ToFloat64(v, 1000)
	if res != 123.456 {
		t.Errorf("Wrong value from Dec128ToFloat64: wait %f, get %f\n", 123.456, res)
	}
	res = Dec128ToFloat64(v, 1000000)
	if res != 0.123456 {
		t.Errorf("Wrong value from Dec128ToFloat64: wait %f, get %f\n", 0.123456, res)
	}
}

func TestDec128ToInt64(t *testing.T) {
	v, _ := primitive.ParseDecimal128("123456")
	res := Dec128ToInt64(v, 100)
	if res != 1234 {
		t.Errorf("Wrong value from Dec128ToInt64: wait %d, get %d\n", 1234, res)
	}
	res = Dec128ToInt64(v, 1000)
	if res != 123 {
		t.Errorf("Wrong value from Dec128ToInt64: wait %d, get %d\n", 123, res)
	}
	res = Dec128ToInt64(v, 1000000)
	if res != 0 {
		t.Errorf("Wrong value from Dec128ToInt64: wait %d, get %d\n", 0, res)
	}
}

func TestPayoutType_GetValue(t *testing.T) {
	dec, _ := primitive.ParseDecimal128("3")
	v := PayoutType{
		Amount: 12345678,
		Decs: dec,
		Sym:    "GOLOS",
	}
	res, _ := v.GetValue()
	if res != 12345.678 {
		t.Errorf("Wrong value from PayoutType.GetValue: wait %f, get %f\n", 12345.678, res)
	}
}