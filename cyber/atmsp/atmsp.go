package atmsp

import (
	"errors"
	"github.com/golang-collections/collections/stack"
	"gitlab.com/stihi/stihi-backend/app"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"math"
	"math/big"
	"reflect"
)

/*
Константин К, [07.10.19 10:14]
[В ответ на Andy Vel]
1. это в коде надо смотреть, проще в тестах, сейчас гляну…
2. у values у каждого элемента есть kind, он определяет, какого вида значение, я ссылку на enum давал

Константин К, [07.10.19 10:19]
переменные setrestorer:
+ t — время в секундах с прошлого использования батарейки
+ p — предыдущий разряд батарейки (0 = полностью заряжена)
ещё был v — вестинг аккаунта, сейчас не поддерживается

Константин К, [07.10.19 10:28]
в setrules во всех трёх функциях одна переменная x:
в mainfunc x = netshares;
в curationfunc x = voteshares;
в timepenalty x = время от создания поста в секундах.

Andy Vel, [07.10.19 14:24]
[В ответ на Константин К]
Спасибо. Но каким образом в функцию передается эта переменная? Она размещается в стэке изначально?

Константин К, [07.10.19 14:31]
[В ответ на Andy Vel]
да, значение с kind=variable — это одна из переменных. в idx — её индекс. для x — 0, для рестореров порядок p,v,t

Константин К, [07.10.19 15:43]
полагаю, происходит следующее:
есть список operators, он обрабатывается последовательно. если встречается push, на стек кладётся value. они тоже обрабатываются последовательно. первый push кладёт первое value.
value.kind говорит, откуда брать число, из nums/consts/vars.
vars в монге не видно, т.к. они подаются на вход. одно значение x или три значения p,v,t

ща проверим на примере:
"func" : {
    "varssize" : 3,
    "operators" : [0,0,3,0,4],
    "values" : [
        {"kind": 1, "idx": 2},
        {"kind": 1, "idx": 0},
        {"kind": 0, "idx": 0},
    ],
    "nums" : [NumberLong(353894400)],
}
operators = [ppush, ppush, pmul, ppush, pdiv], то есть выражение val1*val2/val3
values = ["t","p",353894400]; осталось перевести число из fixed point в понятный вид (разделить на 4096), получается t*p/84600 @UncleAndyV
*/

// Реализация исполнения байткода из БД для формул

const (
	ValueKindNum = int64(0)
	ValueKindVar = int64(1)
	ValueKindConst = int64(2)
	ValueKindUndef = int64(3)
)

type opByteCodeFuncType func(bc *ByteCodeType) error

var (
	operators = []opByteCodeFuncType{
		pPush,		pAdd,		pSub, 		pMul,
		pDiv,		pChs,		pAbs,		pSqrt,
		pPow,		pPow2,		pPow3,		pPow4,
		_pSin,		_pCos,		_pTan,		_pSinh,
		_pTanh,		_pCosh,		_pExp,		_pLog,
		pLog10,		pLog2,		_pAsin,		_pAcos,
		_pAtan,		_pAtan2,	pMax,		pMin,
		pSig,		_pFloor,	_pRound,
	}
)

type ValueIndexType struct {
	Kind		primitive.Decimal128		`bson:"kind"`
	Idx			primitive.Decimal128		`bson:"idx"`
}

type ByteCodeFuncType struct {
	Code		*ByteCodeType				`bson:"code"`
	MaxArg		int64						`bson:"maxarg"`
}

type ByteCodeType struct {
	VarsSize	primitive.Decimal128		`bson:"varssize"`
	Operators	[]primitive.Decimal128		`bson:"operators"`
	Values		[]ValueIndexType			`bson:"values"`
	Nums		[]int64						`bson:"nums"`
	Consts		[]int64						`bson:"consts"`

	vars		[]big.Float
	valIdx		int
	stk			*stack.Stack
}

func (bc *ByteCodeType) SetParams(args ...interface{}) *ByteCodeType {
	bc.vars = make([]big.Float, 0, len(args))

	for _, arg := range args {
		bc.vars = append(bc.vars, _toBig(arg))
	}

	return bc
}

func (bc *ByteCodeType) Run() *big.Float {
	// app.Info.Printf("=====================================================\nRun atmsp:\n%+v\n\n", *bc)

	bc.valIdx = 0
	bc.stk = stack.New()

	for _, op := range bc.Operators {
		// Ищем индекс метода
		_, fnIdx := op.GetBytes()
		fn := operators[fnIdx]

		// fmt.Printf("DBG: op idx: %d\n", fnIdx)

		// Вызываем метод по индексу
		err := fn(bc)
		if err != nil {
			app.Error.Printf("Error in bytecode: %s", err)
		}
	}

	res := _toBig(bc.stk.Pop())
	return &res
}

func (bc *ByteCodeType) Val(v *ValueIndexType) interface{} {
	_, idx := v.Idx.GetBytes()
	_, kind := v.Kind.GetBytes()
	switch int64(kind) {
	case ValueKindNum:
		num := float64(bc.Nums[idx]) / 4096.0
		// app.Debug.Printf("value from num %d (%d) - %f\n", idx, bc.Nums[idx], num)
		return num
	case ValueKindVar:
		// app.Debug.Printf("value from param %d - %s\n", idx, bc.vars[idx].String())
		return bc.vars[idx]
	case ValueKindConst:
		num := float64(bc.Consts[idx]) / 4096.0
		// app.Debug.Printf("value from const %d (%d) - %f\n", idx, bc.Nums[idx], num)
		return num
	case ValueKindUndef:
		// app.Error.Println("Undef value in bytecode!!!")
		return int64(-1)
	}

	app.Error.Println("Not found value in bytecode!!!")
	return int64(-1)
}

/*
	Методы байткода
*/

func pPush(bc *ByteCodeType) error {
	// app.Debug.Printf("push valIdx: %d\n", bc.valIdx)
	// app.Debug.Printf("push pointer: %+v\n", bc.Values[bc.valIdx])
	p := bc.Values[bc.valIdx]
	val := bc.Val(&p)
	// app.Debug.Printf("push val: %+v\n", val)
	v := _toBig(val)
	// app.Debug.Printf("push _toBig: %+v\n", val)

	// app.Debug.Printf("push value: %s\n", v.String())

	bc.stk.Push(v)
	bc.valIdx++

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}

func pAdd(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())
	v2 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("add values: %s + %s\n", v1.String(), v2.String())

	// Складываем числа
	res := new(big.Float).Add(&v1, &v2)

	// Размещаем результат в стэке
	bc.stk.Push(*res)

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}

func pSub(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())
	v2 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("sub values: %s - %s\n", v2.String(), v1.String())

	// Вычитаем числа
	res := new(big.Float).Sub(&v2, &v1)

	// Размещаем результат в стэке
	bc.stk.Push(*res)

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}

func pMul(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())
	v2 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("mul values: %s * %s\n", v1.String(), v2.String())

	// Перемножаем числа
	res := new(big.Float).Mul(&v1, &v2)

	// Размещаем результат в стэке
	bc.stk.Push(*res)

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}
func pDiv(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())
	v2 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("div values: %s / %s\n", v2.String(), v1.String())

	v1float, _ := v1.Float64()
	if v1float == float64(0) {
		return errors.New("divide by zero")
	}

	// Делим числа
	res := new(big.Float).Quo(&v2, &v1)

	// Размещаем результат в стэке
	bc.stk.Push(*res)

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}
func pChs(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("change sign for value: %s\n", v1.String())

	// Меняем знак числа
	res := new(big.Float).Neg(&v1)

	// Размещаем результат в стэке
	bc.stk.Push(*res)

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}
func pAbs(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("abs for value: %s\n", v1.String())

	// Апределяем Abs
	res := new(big.Float).Abs(&v1)

	// Размещаем результат в стэке
	bc.stk.Push(*res)

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}
func pSqrt(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("sqrt for value: %s\n", v1.String())

	// Проверяем что значение не меньше нуля (комплексные числа нам не нужны)
	z, _ := new(big.Float).SetString("0")
	if v1.Cmp(z) < 0 {
		return errors.New("sqrt from negative in bytecode operation")
	}

	// Определяем квадратный корень
	res := new(big.Float).Sqrt(&v1)

	// Размещаем результат в стэке
	bc.stk.Push(*res)

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}
func pPow(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: ppow")
}
func pPow2(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("pow2 for value: %s\n", v1.String())

	// Квадрат
	res := new(big.Float).Mul(&v1, &v1)

	// Размещаем результат в стэке
	bc.stk.Push(*res)

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}
func pPow3(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("pow3 for value: %s\n", v1.String())

	// Куб
	tmp := new(big.Float).Mul(&v1, &v1)
	res := new(big.Float).Mul(tmp, &v1)

	// Размещаем результат в стэке
	bc.stk.Push(*res)

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}
func pPow4(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("pow4 for value: %s\n", v1.String())

	// 4 степень
	tmp1 := new(big.Float).Mul(&v1, &v1)
	tmp2 := new(big.Float).Mul(tmp1, &v1)
	res := new(big.Float).Mul(tmp2, &v1)

	// Размещаем результат в стэке
	bc.stk.Push(*res)

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}
func _pSin(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: psin")
}
func _pCos(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: pcos")
}
func _pTan(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: ptan")
}
func _pSinh(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: psinh")
}
func _pCosh(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: pcosh")
}
func _pTanh(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: ptanh")
}
func _pExp(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: pexp")
}
func _pLog(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: plog")
}
func pLog10(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("log10 for value: %s\n", v1.String())

	// Десятичный логарифм
	f, _ := v1.Float64()

	// Вычисляем логарифм
	l := math.Log10(f)
	res := _toBig(l)

	// Размещаем результат в стэке
	bc.stk.Push(res)

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}
func pLog2(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("log2 for value: %s\n", v1.String())

	// Двоичный логарифм
	f, _ := v1.Float64()

	// Вычисляем логарифм
	l := math.Log2(f)
	res := _toBig(l)

	// Размещаем результат в стэке
	bc.stk.Push(res)

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}
func _pAsin(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: pasin")
}
func _pAcos(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: pacos")
}
func _pAtan(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: patan")
}
func _pAtan2(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: patan2")
}
func pMax(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())
	v2 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("max for values: %s <> %s\n", v1.String(), v2.String())

	// Сравниваем
	if v1.Cmp(&v2) > 0 {
		bc.stk.Push(v1)
	} else {
		bc.stk.Push(v2)
	}

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}
func pMin(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())
	v2 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("min for values: %s <> %s\n", v1.String(), v2.String())

	// Сравниваем
	if v1.Cmp(&v2) < 0 {
		bc.stk.Push(v1)
	} else {
		bc.stk.Push(v2)
	}

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}
func pSig(bc *ByteCodeType) error {
	v1 := _toBig(bc.stk.Pop())

	// app.Debug.Printf("sign for value: %s\n", v1.String())

	// Определяем знак числа
	res := _toBig(v1.Sign())

	// Размещаем результат в стэке
	bc.stk.Push(res)

	// app.Debug.Printf("stack: %+v\n", bc.stk.Peek())

	return nil
}
func _pFloor(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: pfloor")
}
func _pRound(bc *ByteCodeType) error {
	return errors.New("unsupported bytecode operation: pround")
}

func _toBig(v interface{}) big.Float {
	// app.Debug.Printf("_toBig param: %+v\n", v)

	switch v.(type) {
	case big.Float:
		val := v.(big.Float)
		// app.Debug.Printf("_toBig big.Float: %s\n", val.String())
		return val
	case big.Int:
		t := v.(big.Int)
		// app.Debug.Printf("_toBig big.Int: %s\n", new(big.Float).SetInt(&t).String())
		return *new(big.Float).SetInt(&t)
	case int:
		// app.Debug.Printf("_toBig int: %s\n", big.NewFloat(float64(v.(int))).String())
		return *big.NewFloat(float64(v.(int)))
	case int32:
		// app.Debug.Printf("_toBig int32: %s\n", big.NewFloat(float64(v.(int32))))
		return *big.NewFloat(float64(v.(int32)))
	case int64:
		// app.Debug.Printf("_toBig int64: %s\n", big.NewFloat(float64(v.(int64))).String())
		return *big.NewFloat(float64(v.(int64)))
	case float64:
		// app.Debug.Printf("_toBig float64: %s\n", big.NewFloat(v.(float64)).String())
		return *big.NewFloat(v.(float64))
	case float32:
		// app.Debug.Printf("_toBig float32: %s\n", big.NewFloat(float64(v.(float32))).String())
		return *big.NewFloat(float64(v.(float32)))
	case primitive.Decimal128:
		t, _ := new(big.Float).SetString(v.(primitive.Decimal128).String())
		// app.Debug.Printf("_toBig Decimal128: %s\n", t.String())
		return *t
	default:
		app.Error.Printf("Unknown value type for _toBig: %s", reflect.TypeOf(v).String())
	}

	return *big.NewFloat(0)
}
