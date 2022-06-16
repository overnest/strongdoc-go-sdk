package sberr

import (
	"fmt"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func FromError(err error) error {
	if err == nil {
		return nil
	}

	if e, ok := err.(GrpcErr); ok {
		return e
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	return &grpcErr{status: st, details: nil}
}

type GrpcErr interface {
	Code() codes.Code
	Message() string
	Detail() string
	Details() []string
	error
}

type grpcErr struct {
	status  *status.Status
	details []string
	output  string
}

func (e *grpcErr) Code() codes.Code {
	return e.status.Code()
}

func (e *grpcErr) Message() string {
	return e.status.Message()
}

func (e *grpcErr) Detail() string {
	details := e.Details()
	if len(details) > 0 {
		return details[0]
	}
	return ""
}

func (e *grpcErr) Details() []string {
	if len(e.details) > 0 {
		return e.details
	}

	e.details = make([]string, len(e.status.Details()))
	for i, detail := range e.status.Details() {
		switch t := detail.(type) {
		case *errdetails.LocalizedMessage:
			e.details[i] = t.Message
		default:
			e.details[i] = ""
		}
	}

	return e.details
}

func (e *grpcErr) Error() string {
	if len(e.output) > 0 {
		return e.output
	}

	e.output = fmt.Sprintf("gRPC error: code = %v, msg = %v", e.Code(), e.Message())

	details := e.Details()
	if len(details) > 0 {
		var sb strings.Builder
		for _, detail := range details {
			sb.WriteString(fmt.Sprintf("%v\n", detail))
		}
		e.output = fmt.Sprintf("%v\n%v", e.output, sb.String())
	}

	return e.output
}
