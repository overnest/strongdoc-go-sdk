package client

import (
	"context"
	"flag"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	addr = flag.String("addr", "api.strongsalt.com", "The StrongSalt API server")
	port = flag.String("port", "9090", "The port to connect to")
	cert = flag.String("cert", "./ssca.cert.pem", "The root certificate used to connect to the server")
	//addr = flag.String("addr", "localhost", "The address of the server to connect to")
	//cert = flag.String("cert", "./localhost.crt", "The root certificate used to connect to the server")
)

// TokenAuth is the auth token used to call gRPC APIs that requires auth
type TokenAuth struct {
	Token string
}

// GetRequestMetadata gets the request metadata
func (t *TokenAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.Token,
	}, nil
}

// RequireTransportSecurity requires transport authority
func (t *TokenAuth) RequireTransportSecurity() bool {
	return true
}

// NoAuth is the auth token used to call gRPC APIs that does not require auth
type NoAuth struct{}

// GetRequestMetadata gets the request metadata
func (t *NoAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	// No additional values in request headers
	return map[string]string{}, nil
}

// RequireTransportSecurity requires transport authority
func (t *NoAuth) RequireTransportSecurity() bool {
	return true
}

// ConnectToServerNoAuth creates a connection to the server without any authorization
func ConnectToServerNoAuth() (conn *grpc.ClientConn, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Create the client TLS credentials
	creds, err := credentials.NewClientTLSFromFile(*cert, "")
	if err != nil {
		log.Fatalf("could not load tls cert: %s", err)
	}

	// Initiate a connection with the server
	return grpc.DialContext(ctx, *addr+":"+*port,
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(&NoAuth{}),
	)
}

// ConnectToServerWithAuth creates a connection to the server without any authorization
func ConnectToServerWithAuth(token string) (conn *grpc.ClientConn, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Create the client TLS credentials
	creds, err := credentials.NewClientTLSFromFile(*cert, "")
	if err != nil {
		log.Fatalf("could not load tls cert: %s", err)
	}

	// Initiate a connection with the server
	return grpc.DialContext(ctx, *addr+":"+*port,
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(&TokenAuth{Token: token}),
	)
}
