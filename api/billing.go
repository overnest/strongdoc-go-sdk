package api

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
	"github.com/overnest/strongdoc-go-sdk/utils"
)

// BillingDetails stores the billing details for the organization
type BillingDetails struct {
	// Start of the requested billing period
	PeriodStart time.Time
	// End of the requested billing period
	PeriodEnd time.Time
	// Total cost incurred during the requested billing period
	TotalCost float64
	// Usage and cost breakdown for stored documents
	Documents *DocumentCosts
	// Usage and cost breakdown for stored search indices
	Search *SearchCosts
	// Usage and cost breakdown for used traffic
	Traffic *TrafficCosts
}

// DocumentCosts stores the document cost portion of the bill
type DocumentCosts struct {
	// Cost of document storage incurred during a billing period
	Cost float64
	// Size of documents stored during a billing period (in MBhours)
	Size float64
	// Cost tier reached for document storage during a billing period
	Tier string
}

// SearchCosts stores the search cost portion of the bill
type SearchCosts struct {
	// Cost of search index storage incurred during a billing period
	Cost float64
	// Size of search indices stored during a billing period (in MBhours)
	Size float64
	// Cost tier reached for search index storage during a billing period
	Tier string
}

// TrafficCosts stores the traffic coast portion of the bill
type TrafficCosts struct {
	// Cost of network traffic incurred during a billing period
	Cost float64
	// Size of incoming requests during a billing period (in MB)
	Incoming float64
	// Size of outgoing requests during a billing period (in MB)
	Outgoing float64
	// Cost tier reached for network traffic during a billing period
	Tier string
}

//GetBillingDetails list all items of the cost breakdown and also other details such as the billing frequency
func GetBillingDetails() (bill *BillingDetails, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}
	req := &proto.GetBillingDetailsReq{}
	res, err := sdc.GetBillingDetails(context.Background(), req)
	if err != nil {
		return
	}

	billing, err := utils.ConvertStruct(res, &BillingDetails{})
	if err != nil {
		return nil, err
	}

	return billing.(*BillingDetails), nil
}

// BillingFrequency shows the billing frequency information
type BillingFrequency struct {
	// Billing frequency
	Frequency proto.TimeInterval
	// Start fo billing frequency validity
	ValidFrom *time.Time
	// End of billing frequency validity
	ValidTo *time.Time
}

//GetBillingFrequencyList obtains the list of billing frequencies (past, current and future)
func GetBillingFrequencyList() ([]*BillingFrequency, error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return nil, err
	}

	req := &proto.GetBillingFrequencyListReq{}
	resp, err := sdc.GetBillingFrequencyList(context.Background(), req)
	if err != nil {
		return nil, err
	}

	freqList, err := utils.ConvertStruct(resp.GetBillingFrequencyList(), []*BillingFrequency{})
	if err != nil {
		return nil, err
	}

	return *freqList.(*[]*BillingFrequency), nil
}

//SetNextBillingFrequency changes the next billing frequency
func SetNextBillingFrequency(freq proto.TimeInterval, validFrom time.Time) (*BillingFrequency, error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return nil, err
	}

	from, err := ptypes.TimestampProto(validFrom)
	if err != nil {
		return nil, err
	}

	req := &proto.SetNextBillingFrequencyReq{Frequency: freq, ValidFrom: from}
	resp, err := sdc.SetNextBillingFrequency(context.Background(), req)
	if err != nil {
		return nil, err
	}

	frequency, err := utils.ConvertStruct(resp.GetNextBillingFrequency(), &BillingFrequency{})
	if err != nil {
		return nil, err
	}

	return frequency.(*BillingFrequency), nil
}

// LargeTraffic contains the large traffic data
type LargeTraffic struct {
	// Details of large traffic events
	LargeTraffic []*TrafficDetail
	// Start of the requested billing period
	PeriodStart time.Time
	// End of the requested billing period
	PeriodEnd time.Time
}

// TrafficDetail contains the traffic detail
type TrafficDetail struct {
	// Timestamp of the large traffic event
	Time time.Time
	// The ID of the user who made the request
	UserID string
	// HTTP method of the request
	Method string
	// URI called by the request
	URI string
	// Size of the request (in MB)
	Incoming float64
	// Size of the response (in MB)
	Outgoing float64
}

//GetLargeTraffic obtains the list of large traffic usages
func GetLargeTraffic(at time.Time) (*LargeTraffic, error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return nil, err
	}

	atproto, err := ptypes.TimestampProto(at)
	if err != nil {
		return nil, err
	}

	req := &proto.GetLargeTrafficReq{At: atproto}
	resp, err := sdc.GetLargeTraffic(context.Background(), req)
	if err != nil {
		return nil, err
	}

	traffic, err := utils.ConvertStruct(resp, &LargeTraffic{})
	if err != nil {
		return nil, err
	}

	return traffic.(*LargeTraffic), nil
}
