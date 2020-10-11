package atmsp

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
)

// res = x * 38678321 / 279892 + 43210
func TestCheckWork(t *testing.T) {
	d128_0, _ := primitive.ParseDecimal128("0")
	d128_1, _ := primitive.ParseDecimal128("1")
	d128_2, _ := primitive.ParseDecimal128("2")
	d128_3, _ := primitive.ParseDecimal128("3")
	d128_4, _ := primitive.ParseDecimal128("4")
	machine := ByteCodeType{
		VarsSize:  d128_1,
		Operators: []primitive.Decimal128{
			d128_0,
			d128_0,
			d128_3,
			d128_0,
			d128_4,
			d128_0,
			d128_1,
		},
		Values:    []ValueIndexType{
			{
				Kind: d128_1,
				Idx:  d128_0,
			},
			{
				Kind: d128_0,
				Idx: d128_0,
			},
			{
				Kind: d128_0,
				Idx: d128_1,
			},
			{
				Kind: d128_0,
				Idx: d128_2,
			},
		},
		Nums:      []int64{
			38678321 * 4096,
			279892 * 4096,
			43210 * 4096,
		},
		Consts:    []int64{},
	}

	fmt.Printf("Machine: %+v\n", machine)

	machine.SetParams(0)
	res0 := machine.Run()
	if res0.String() != "43210" {
		t.Errorf("Error calc with arg 0: get %s, expected %s", res0.String(), "43210")
	}

	machine.SetParams(1)
	res1 := machine.Run()
	if res1.String() != "43348.19016" {
		t.Errorf("Error calc with arg 1: get %s, expected %s", res1.String(), "43348.19016")
	}

	machine.SetParams(123)
	res123 := machine.Run()
	if res123.String() != "60207.39" {
		t.Errorf("Error calc with arg 1: get %s, expected %s", res123.String(), "60207.39")
	}

	machine.SetParams(67381001)
	resBig := machine.Run()
	if resBig.String() != "9311434697" {
		t.Errorf("Error calc with arg 1: get %s, expected %s", resBig.String(), "9311434697")
	}
}

func TestCheckSharesFN(t *testing.T) {
	d128_0, _ := primitive.ParseDecimal128("0")
	d128_1, _ := primitive.ParseDecimal128("1")
	d128_2, _ := primitive.ParseDecimal128("2")
	d128_3, _ := primitive.ParseDecimal128("3")
	d128_4, _ := primitive.ParseDecimal128("4")
	machine := ByteCodeType{
		VarsSize:  d128_1,
		Operators: []primitive.Decimal128{
			d128_0,
			d128_0,
			d128_1,
			d128_0,
			d128_0,
			d128_1,
			d128_4,
			d128_0,
			d128_0,
			d128_4,
			d128_3,
		},
		Values:    []ValueIndexType{
			{
				Kind: d128_1,
				Idx:  d128_0,
			},
			{
				Kind: d128_0,
				Idx: d128_0,
			},
			{
				Kind: d128_1,
				Idx: d128_0,
			},
			{
				Kind: d128_0,
				Idx: d128_1,
			},
			{
				Kind: d128_1,
				Idx: d128_0,
			},
			{
				Kind: d128_0,
				Idx: d128_2,
			},
		},
		Nums:      []int64{
			16384000000000000,
			32768000000000000,
			16777216,
		},
		Consts:    []int64{},
	}

	machine.SetParams(14930561479671)
	res := machine.Run()
	if res.String() != "3009296628" {
		t.Errorf("Error calc with arg 0: get %s, expected %s", res.String(), "3009296628")
	}
}
