package api

import (
	"context"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/overnest/strongdoc-go/client"
	"github.com/overnest/strongdoc-go/proto"
	"log"
)

type BillingDetails struct {
	protoBillingDetails *proto.GetBillingDetailsResponse
}

type BillingPeriod struct {
	protoBillingPeriod *proto.BillingPeriod
}

type BillingDocuments struct {
	protoBillingDocument *proto.Documents
}

type BillingIndex struct {
	protoBillingIndex *proto.Index
}

type BillingTraffic struct {
	protoBillingTraffic *proto.Traffic
}

func (bd *BillingDetails) CurrentPeriod() (CurrentPeriod *BillingPeriod, err error) {
	if bd.protoBillingDetails == nil {
		return nil, err
	}
	cPeriod := BillingPeriod{bd.protoBillingDetails.CurrentPeriod}
	return &cPeriod, nil
}

func (bd *BillingDetails) TotalCost() (TotalCost int32, err error) {
	if bd.protoBillingDetails == nil {
		return 0, err
	}
	return bd.protoBillingDetails.TotalCost, nil
}

func (bd *BillingDetails) Documents() (bdoc *BillingDocuments, err error) {
	if bd.protoBillingDetails == nil {
		return nil, err
	}
	bdoc = &BillingDocuments{bd.protoBillingDetails.Documents}
	return bdoc, nil
}

func (d *BillingDocuments) Cost() int32 {
	return d.protoBillingDocument.Cost
}

func (d *BillingDocuments) Size() float64 {
	return d.protoBillingDocument.Size
}

func (bd *BillingDetails) Index() (bi *BillingIndex, err error) {
	if bd.protoBillingDetails == nil {
		return nil, err
	}
	bi = &BillingIndex{bd.protoBillingDetails.Index}
	return bi, nil
}

func (i *BillingIndex) Cost() int32 {
	return i.protoBillingIndex.Cost
}

func (i *BillingIndex) Size() int64 {
	return i.protoBillingIndex.Size
}

func (bd *BillingDetails) NextPeriod() (bp *BillingPeriod, err error) {
	if bd.protoBillingDetails == nil {
		return nil, err
	}
	bp = &BillingPeriod{bd.protoBillingDetails.NextPeriod}
	return bp, nil
}

func (bp *BillingPeriod) Frequency() string {
	return bp.protoBillingPeriod.Frequency.String()
}

func (bp *BillingPeriod) PeriodStart() *timestamp.Timestamp {
	return bp.protoBillingPeriod.PeriodStart
}

func (bp *BillingPeriod) PeriodEnd() *timestamp.Timestamp {
	return bp.protoBillingPeriod.PeriodEnd
}

func (bd *BillingDetails) Traffic() (bt *BillingTraffic, err error) {
	if bd.protoBillingDetails == nil {
		return nil, err
	}
	bt = &BillingTraffic{bd.protoBillingDetails.Traffic}
	return bt, nil
}

func (t *BillingTraffic) Cost() int32 {
	return t.protoBillingTraffic.Cost
}

func (t *BillingTraffic) Incoming() float64 {
	return t.protoBillingTraffic.Incoming
}

func (t *BillingTraffic) Outgoing() float64 {
	return t.protoBillingTraffic.Outgoing
}

func Billing(token string) (bd BillingDetails) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)
	req := &proto.GetBillingDetailsRequest{}
	res, err := authClient.GetBillingDetails(context.Background(), req)
	//authClient.GetBilling
	return BillingDetails{res}
}

