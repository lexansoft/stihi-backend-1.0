package cyberdb

import (
	"gitlab.com/stihi/stihi-backend/cyber/atmsp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"math"
	"math/big"
	"strconv"
)

const (
	CyberDBName			= "_CYBERWAY_"
	TokensDBName		= "_CYBERWAY_cyber_token"
	PublishDBName 		= "_CYBERWAY_gls_publish"
	ChargeDBName 		= "_CYBERWAY_gls_charge"
	ControlDBName		= "_CYBERWAY_gls_ctrl"
	EmmitDBName			= "_CYBERWAY_gls_emit"
	MemoDBName			= "_CYBERWAY_gls_memo"
	ReferralDBName		= "_CYBERWAY_gls_referral"
	SocialDBName		= "_CYBERWAY_gls_social"
	VestingDBName		= "_CYBERWAY_gls_vesting"
)

// _CYBERWAY_gls_publish
type VoteType struct {
	ObjectId		primitive.ObjectID		`bson:"_id"`
	Id				primitive.Decimal128	`bson:"id"`

	MessageId		primitive.Decimal128	`bson:"message_id"`
	Voter			string					`bson:"voter"`
	Weight			int64					`bson:"weight"`
	Time			primitive.Decimal128	`bson:"time"`
	Count			primitive.Decimal128	`bson:"count"`
	Delegators		[]DelegatorType	`bson:"delegators"`
	CuratorsW		int64			`bson:"curatorsw"`
	RShares			int64			`bson:"rshares"`
	PaidAmount		int64			`bson:"paid_amount"`
	Service			ServiceType		`bson:"_SERVICE_"`
}

type ServiceType struct {
	Scope			string			`bson:"scope"`
	Rev				int64			`bson:"rev"`
	Payer			string			`bson:"payer"`
	Size			int				`bson:"size"`
	Ram				bool			`bson:"ram"`
}

type DelegatorType struct {
	Delegator		string					`bson:"delegator"`
	InterestRate	primitive.Decimal128	`bson:"interest_rate"`
}

type MessageType struct {
	ObjectId		primitive.ObjectID		`bson:"_id"`
	Id				primitive.Decimal128	`bson:"id"`

	Author			string					`bson:"author"`
	Date			primitive.Decimal128	`bson:"date"`
	PoolDate		primitive.Decimal128	`bson:"pool_date"`
	TokenProp		primitive.Decimal128	`bson:"tokenprop"`
	Beneficiaries	[]interface{}			`bson:"beneficiaries"`
	RewardWeight	primitive.Decimal128	`bson:"rewardweight"`
	CuratorsPrcnt	primitive.Decimal128	`bson:"curators_prcnt"`
	CashoutTime		primitive.Decimal128	`bson:"cashout_time"`
	PaidAmount		int64					`bson:"paid_amount"`

	State			MessageStateType		`bson:"state"`
	MessageReward	PayoutType				`bson:"mssg_reward"`
	MaxPayout		PayoutType				`bson:"max_payout"`

	Service			ServiceType				`bson:"_SERVICE_"`
}

type MessageStateType struct {
	NetShares		int64			`bson:"netshares"`
	VoteShares		int64			`bson:"voteshares"`
	SumCuratorsW	int64			`bson:"sumcuratorsw"`
}

type PayoutType struct {
	Amount			int64					`bson:"_amount"`
	Decs			primitive.Decimal128	`bson:"_decs"`
	Sym				string					`bson:"_sym"`
}

func (p PayoutType) GetValue() (float64, error) {
	dec, err := strconv.ParseInt(p.Decs.String(), 10, 64)
	if err != nil {
		return 0, err
	}
	decF := math.Pow(10, float64(dec))

	return float64(p.Amount)/decF, nil
}

func (p PayoutType) GetBigValue() (*big.Float, error) {
	dec, err := strconv.ParseInt(p.Decs.String(), 10, 64)
	if err != nil {
		return nil, err
	}
	decF := math.Pow(10, float64(dec))

	res := big.NewFloat(0).Quo(big.NewFloat(float64(p.Amount)), big.NewFloat(decF))

	return res, nil
}

type PermlinkType struct {
	ObjectId		primitive.ObjectID		`bson:"_id"`
	Id				primitive.Decimal128	`bson:"id"`

	Value			string					`bson:"value"`

	ParentAcc		string					`bson:"parentacc"`
	ParentId		primitive.Decimal128	`bson:"parent_id"`
	Level			primitive.Decimal128	`bson:"level"`
	ChildCount		primitive.Decimal128	`bson:"childcount"`

	Service			ServiceType				`bson:"_SERVICE_"`
}

type RewardPoolType struct {
	ObjectId		primitive.ObjectID		`bson:"_id"`
	Created			primitive.Decimal128	`bson:"created"`

	Rules			RewardPoolRulesType		`bson:"rules"`
	State			RewardPoolStateType		`bson:"state"`

	Service			ServiceType				`bson:"_SERVICE_"`
}

type RewardPoolStateType struct {
	Msgs			primitive.Decimal128	`bson:"msgs"`
	Funds			PayoutType				`bson:"funds"`
	RShares			RSharesType				`bson:"rshares"`
	RSharesFN		RSharesType				`bson:"rsharesfn"`
}

type RSharesType struct {
	Binary			[]byte					`bson:"_binary"`
	String			string					`bson:"_string"`
}

type RewardPoolRulesType struct {
	MainFunc		*atmsp.ByteCodeFuncType	`bson:"mainfunc"`
	CurationFunc	*atmsp.ByteCodeFuncType	`bson:"curationfunc"`
	TimePenalty		*atmsp.ByteCodeFuncType	`bson:"timepenalty"`
	MaxTokenProp	primitive.Decimal128	`bson:"maxtokenprop"`
}

// _CYBERWAY_cyber_token
type TokenAccountType struct {
	ObjectId		primitive.ObjectID		`bson:"_id"`

	Balance			PayoutType				`bson:"balance"`
	Payments		PayoutType				`bson:"payments"`

	Service			ServiceType				`bson:"_SERVICE_"`
}

// _CYBERWAY_gls_vesting
type VestingAccountType struct {
	ObjectId		primitive.ObjectID		`bson:"_id"`

	Vesting			PayoutType				`bson:"vesting"`
	Delegated		PayoutType				`bson:"delegated"`
	Received		PayoutType				`bson:"received"`
	UnlockedLimit	PayoutType				`bson:"unlocked_limit"`
	Delegators		primitive.Decimal128	`bson:"delegators"`

	Service			ServiceType				`bson:"_SERVICE_"`
}

type VestingStatType struct {
	ObjectId		primitive.ObjectID		`bson:"_id"`

	Supply			PayoutType				`bson:"supply"`
	NotifyAcc		string					`bson:"notify_acc"`

	Service			ServiceType				`bson:"_SERVICE_"`
}

// _CYBERWAY_
type PermissionType struct {
	ObjectId		primitive.ObjectID			`bson:"_id"`

	Id				primitive.Decimal128		`bson:"id"`
	UsageId			primitive.Decimal128		`bson:"usage_id"`
	Parent			primitive.Decimal128		`bson:"parent"`
	Owner			string						`bson:"owner"`
	Name			string						`bson:"name"`
	LastUpdated		string						`bson:"last_updated"`
	Auth			AccountAuthType				`bson:"auth"`

	Service			ServiceType					`bson:"_SERVICE_"`
}

type AccountAuthType struct {
	Threshold		primitive.Decimal128		`bson:"threshold"`
	Waits			[]WaitsType					`bson:"waits"`
	Accounts		[]AccountLineType			`bson:"accounts"`
	Keys			[]KeyType					`bson:"keys"`
}

type WaitsType struct {
	WaitSec			primitive.Decimal128	`bson:"wait_sec"`
	Weight			primitive.Decimal128	`bson:"weight"`
}

type AccountPermissionType struct {
	Permission 		string		`json:"permission"`
	Actor			string		`json:"actor"`
}

type AccountLineType struct {
	Permission		AccountPermissionType	`bson:"permission"`
	Weight			primitive.Decimal128	`bson:"weight"`
}

type KeyType struct {
	Weight			primitive.Decimal128	`bson:"weight"`
	Key				string					`bson:"key"`
}

type UserNameType struct {
	ObjectId		primitive.ObjectID			`bson:"_id"`

	Id				primitive.Decimal128		`bson:"id"`
	Owner			string						`bson:"owner"`
	Scope			string						`bson:"scope"`
	Name			string						`bson:"name"`

	Service			ServiceType					`bson:"_SERVICE_"`
}

type ChargeBalanceType struct {
	ObjectId		primitive.ObjectID			`bson:"_id"`

	ChargeSymbol	primitive.Decimal128		`bson:"charge_symbol"`
	TokenCode		string						`bson:"token_code"`
	ChargeId		primitive.Decimal128		`bson:"charge_id"`
	LastUpdate		primitive.Decimal128		`bson:"last_update"`
	Value			int64						`bson:"value"`

	Service			ServiceType					`bson:"_SERVICE_"`
}

type ChargeRestorerType struct {
	ObjectId		primitive.ObjectID			`bson:"_id"`

	ChargeSymbol	primitive.Decimal128		`bson:"charge_symbol"`
	TokenCode		string						`bson:"token_code"`
	ChargeId		primitive.Decimal128		`bson:"charge_id"`
	Func			*atmsp.ByteCodeType			`bson:"func"`
	MaxPrev			int64						`bson:"max_prev"`
	MaxVesting		int64						`bson:"max_vesting"`
	MaxElapsed		int64						`bson:"max_elapsed"`

	Service			ServiceType					`bson:"_SERVICE_"`
}


func Dec128ToInt64L(v primitive.Decimal128) int64 {
	t, _ := new(big.Float).SetString(v.String())
	res, _ := t.Int64()
	return res
}

func Dec128ToInt64(v primitive.Decimal128, devider int64) int64 {
	t, _ := new(big.Float).SetString(v.String())
	d := big.NewFloat(0).Quo(t, big.NewFloat(float64(devider)))

	res, _ := d.Int64()

	return res
}

func Dec128ToFloat64(v primitive.Decimal128, devider int64) float64 {
	t, _ := new(big.Float).SetString(v.String())
	d := big.NewFloat(0).Quo(t, big.NewFloat(float64(devider)))

	res, _ := d.Float64()

	return res
}


func Dec128ToBigFloat(v primitive.Decimal128, devider int64) *big.Float {
	t, _ := new(big.Float).SetString(v.String())
	d := big.NewFloat(0).Quo(t, big.NewFloat(float64(devider)))
	return d
}

func ListInt64ToBsonA(list *[]int64) bson.A {
	uniqList := make([]int64, 0, len(*list))
	already := make(map[int64]bool)
	for _, id := range *list {
		if !already[id] && id != 1 {
			uniqList = append(uniqList, id)
			already[id] = true
		}
	}

	b := make(bson.A, len(uniqList))
	for i := 0; i < len(uniqList); i++ {
		b[i] = uniqList[i]
	}
	return b
}
