package client

import (
	"context"
)

type grpcAuthCred struct {
	token string
}

func (t *grpcAuthCred) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.token,
	}, nil
}

func (t *grpcAuthCred) RequireTransportSecurity() bool {
	return true
}

type grpcNoAuthCred struct{}

func (t *grpcNoAuthCred) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	// No additional values in Req headers
	return map[string]string{}, nil
}

func (t *grpcNoAuthCred) RequireTransportSecurity() bool {
	return true
}
