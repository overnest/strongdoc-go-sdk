package utils

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/overnest/strongdoc-go-sdk/proto"
	assert "github.com/stretchr/testify/require"
)

type AccountInfo struct {
	DoesNotExist    string
	OrgID           *string
	Subscription    *Subscription
	Payments        []*Payment
	OrgAddress      string
	MultiLevelShare bool
	SharableOrgs    []string
}

type Subscription struct {
	Type   string
	Status string
}

type Payment struct {
	BilledAt    *time.Time
	PeriodStart *time.Time
	PeriodEnd   *time.Time
	Amount      float64
	Status      string
}

func TestConvert(t *testing.T) {
	gAccount := &proto.GetAccountInfoResp{
		OrgID: "testorg",
		Subscription: &proto.Subscription{
			Type:   "subscription type",
			Status: "subscription status",
		},
		Payments: []*proto.Payment{
			&proto.Payment{
				BilledAt:    ptypes.TimestampNow(),
				PeriodStart: ptypes.TimestampNow(),
				PeriodEnd:   ptypes.TimestampNow(),
				Amount:      100,
				Status:      "payment status",
			},
		},
		OrgAddress:      "some address",
		MultiLevelShare: true,
		SharableOrgs:    []string{"shareorg1"}}

	cAccount, err := ConvertStruct(gAccount, &AccountInfo{})
	assert.NoError(t, err)
	fmt.Println("cAccount:", cAccount)
}

func TestConvertScalar(t *testing.T) {
	var res interface{}
	var err error
	mystring := "mystring"

	res, err = ConvertStruct(nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, res)

	res, err = ConvertStruct(nil, mystring)
	assert.NoError(t, err)
	assert.Equal(t, res, mystring)

	res, err = ConvertStruct(mystring, nil)
	assert.NoError(t, err)
	assert.Nil(t, res)

	res, err = ConvertStruct((*string)(nil), (*string)(nil))
	assert.NoError(t, err)
	assert.Nil(t, res)

	res, err = ConvertStruct(&mystring, mystring)
	assert.NoError(t, err)
	fmt.Println(res)
}

type TimeTestGo struct {
	One   time.Time
	Two   time.Time
	Three *time.Time
	Four  *time.Time
}

type TimeTestGrpc struct {
	One   timestamp.Timestamp
	Two   *timestamp.Timestamp
	Three timestamp.Timestamp
	Four  *timestamp.Timestamp
}

func TestConvertTime(t *testing.T) {
	timeGo := &TimeTestGo{}
	timeGrpc := &TimeTestGrpc{
		One:   *ptypes.TimestampNow(),
		Two:   ptypes.TimestampNow(),
		Three: *ptypes.TimestampNow(),
		Four:  ptypes.TimestampNow(),
	}

	timeNew, err := ConvertStruct(timeGrpc, nil)
	assert.NoError(t, err)
	assert.Nil(t, timeNew)
	timeNew = nil

	timeNew, err = ConvertStruct(timeGrpc, (*TimeTestGo)(nil))
	assert.NoError(t, err)
	fmt.Println("goTime:", timeNew)
	timeNew = nil

	timeNew, err = ConvertStruct(timeGrpc, timeGo)
	assert.NoError(t, err)
	fmt.Println("goTime:", timeNew)
	timeNew = nil
}

type StructTestGrpc struct {
	One   proto.Subscription
	Two   *proto.Subscription
	Three proto.Subscription
	Four  *proto.Subscription
}

type StructTestGo struct {
	One   *Subscription
	Two   *Subscription
	Three Subscription
	Four  Subscription
}

func TestConvertStruct(t *testing.T) {
	sub := proto.Subscription{
		Type:   "subscription type",
		Status: "subscription status",
	}

	sgo := &StructTestGo{}
	sgrpc := &StructTestGrpc{
		One:   sub,
		Two:   &sub,
		Three: sub,
		Four:  &sub,
	}
	structNew, err := ConvertStruct(sgrpc, sgo)
	assert.NoError(t, err)
	fmt.Println("goStruct:", structNew)
}

type SliceTestGrpc struct {
	One   []*proto.Subscription
	Two   []proto.Subscription
	Three []*proto.Subscription
}

type SliceTestGo struct {
	One   []Subscription
	Two   []*Subscription
	Three []Subscription
}

func TestConvertSlice(t *testing.T) {
	sgrpc := &SliceTestGrpc{
		One: []*proto.Subscription{
			&proto.Subscription{Type: "One 1", Status: "One 1"},
			&proto.Subscription{Type: "One 2", Status: "One 2"},
		},
		Two: []proto.Subscription{
			proto.Subscription{Type: "Two 1", Status: "Two 1"},
			proto.Subscription{Type: "Two 2", Status: "Two 2"},
		},
		Three: nil,
	}

	structNew, err := ConvertStruct(sgrpc, &SliceTestGo{})
	assert.NoError(t, err)
	fmt.Println("goStruct:", structNew)
}

type MapTestGrpc struct {
	One   map[string]*proto.Subscription
	Two   map[string]proto.Subscription
	Three map[string]*proto.Subscription
}

type MapTestGo struct {
	One   map[string]Subscription
	Two   map[string]*Subscription
	Three map[string]*Subscription
}

func TestConvertMap(t *testing.T) {
	mgrpc := &MapTestGrpc{
		One: map[string]*proto.Subscription{
			"kOne 1": &proto.Subscription{Type: "One 1", Status: "One 1"},
			"kOne 2": &proto.Subscription{Type: "One 2", Status: "One 2"},
		},
		Two: map[string]proto.Subscription{
			"kTwo 1": proto.Subscription{Type: "Two 1", Status: "Two 1"},
			"kTwo 2": proto.Subscription{Type: "Two 2", Status: "Two 2"},
		},
		Three: nil,
	}

	structNew, err := ConvertStruct(mgrpc, &MapTestGo{})
	assert.NoError(t, err)
	fmt.Println("goStruct:", structNew)

	toMap := map[string]*Subscription{}
	fromMap := map[string]*proto.Subscription{
		"kOne 1": &proto.Subscription{Type: "One 1", Status: "One 1"},
		"kOne 2": &proto.Subscription{Type: "One 2", Status: "One 2"},
	}
	structNew, err = ConvertStruct(fromMap, toMap)
	assert.NoError(t, err)
	fmt.Println("goStruct:", *structNew.(*map[string]*Subscription))
}

func TestGoReflection(t *testing.T) {
	sub := proto.Subscription{
		Type:   "subscription type",
		Status: "subscription status",
	}

	test := reflect.New(reflect.ValueOf(StructTestGo{}).Type()).Elem()
	printValue(test)

	for i := 0; i < test.NumField(); i++ {
		fmt.Println("------------")
		field := test.Field(i)
		printValue(field)

		var sub reflect.Value
		if field.Kind() == reflect.Ptr {
			sub = reflect.New(field.Type().Elem()).Elem()
		} else {
			sub = reflect.New(field.Type()).Elem()
		}
		printValue(sub)

		for j := 0; j < reflect.Indirect(sub).NumField(); j++ {
			subField := sub.Field(j)
			subField.SetString(fmt.Sprintf("value %v", i))
		}

		subi := sub.Addr().Interface()
		subnew := reflect.ValueOf(subi)
		printValue(subnew)

		if field.Kind() == reflect.Ptr {
			field.Set(subnew)
		} else {
			field.Set(subnew.Elem())
		}
	}

	// printValue(test)
	testi := test.Interface()
	fmt.Println(testi)

	printValue(reflect.ValueOf(sub))
	printValue(reflect.ValueOf(&sub))
	printValue(reflect.Indirect(reflect.ValueOf(sub)))
	printValue(reflect.Indirect(reflect.ValueOf(&sub)))
	printValue(reflect.ValueOf((*proto.Subscription)(nil)))
	printValue(reflect.ValueOf(nil))
	printValue(reflect.New(reflect.ValueOf(sub).Type()).Elem())
}
