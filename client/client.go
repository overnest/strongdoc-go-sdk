package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/overnest/strongdoc-go-sdk/api/apidef"
	"github.com/overnest/strongdoc-go-sdk/proto"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"github.com/overnest/strongsalt-crypto-go/pake/srp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var clientInit uint32 = 0
var clientMutex sync.Mutex

const MAX_CONCURRENT_CONNECTIONS = 10 // max number of current connections

type locationConfig struct {
	HostPort string
	Cert     string
}

// ServiceLocation specifies the location of the StrongDoc service
type ServiceLocation string

const (
	// DEFAULT is the default production service location
	DEFAULT ServiceLocation = "DEFAULT"
	// SANDBOX is the sandbox testing location
	SANDBOX ServiceLocation = "SANDBOX"
	// QA is the QA service used only for testing
	QA ServiceLocation = "QA"
	// LOCAL is the local service location used only for testing
	LOCAL ServiceLocation = "LOCAL"
	// unset specifies that the service location is not set
	unset ServiceLocation = "UNSET"
)

var serviceLocations = map[ServiceLocation]locationConfig{
	DEFAULT: {"api.strongsalt.com:9090", "./certs/ssca.cert.pem"},
	SANDBOX: {"api.sandbox.strongsalt.com:9090", "./certs/ssca.cert.pem"},
	QA:      {"api.strongsaltqa.com:9090", "./certs/ssca.cert.pem"},
	LOCAL:   {"localhost:9090", "./certs/localhost.cert.pem"},
}

type connPool struct {
	lock  sync.Mutex
	conns []*grpc.ClientConn // all connections
	idx   int                // index
}

// initialize connection pool
func initConnPool(capacity int, authToken, hostPort, cert string) (*connPool, error) {
	conns := make([]*grpc.ClientConn, capacity)
	for i := 0; i < len(conns); i++ {
		conn, err := getAuthConn(authToken, hostPort, cert)
		if err != nil {
			return nil, err
		}
		conns[i] = conn
	}

	return &connPool{
		conns: conns,
		idx:   0,
	}, nil
}

// get usable connection
func (q *connPool) getConn() *grpc.ClientConn {
	q.lock.Lock()
	defer q.lock.Unlock()
	conn := q.conns[q.idx]
	q.idx = (q.idx + 1) % len(q.conns)
	return conn
}

// close all connections
func (q *connPool) closeAll() error {
	q.lock.Lock()
	defer q.lock.Unlock()
	for _, conn := range q.conns {
		err := conn.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

type strongDocClientObj struct {
	location     ServiceLocation
	noAuthConn   *grpc.ClientConn
	authConnPool *connPool
	authToken    string
	user         *apidef.User
}

// Singletons
var serviceLocation ServiceLocation = unset
var client StrongDocClient = nil

// StrongDocClient encapsulates the client object that allows connection to the remote service
type StrongDocClient interface {
	Login(userID, password string) error
	// Logout() (string, error)
	// NewAuthSession(password string) (*AuthSession, error)
	GetNoAuthConn() *grpc.ClientConn
	GetAuthConnPool() *connPool
	GetGrpcClient() proto.StrongDocServiceClient
	Close()
	// UserEncrypt([]byte) ([]byte, error)
	// UserEncryptBase64([]byte) (string, error)
	// UserDecrypt([]byte) ([]byte, error)
	// UserDecryptBase64(string) ([]byte, error)
	// GetUserKeyID() string
	// GetUserID() string
	// ChangePassword(string, string) error
}

// type AuthSession struct {
// 	authID      string
// 	authType    proto.AuthType
// 	authVersion int32
// 	// Login
// 	loginResp *proto.LoginResp
// 	// SRP
// 	srpClient    *srp.Client
// 	srpSharedKey *ssc.StrongSaltKey
// }

// Login attempts a log in. If successful, it generates an authenticatecd GRPC connection
func (c *strongDocClientObj) Login(userEmailOrID, password string) error {
	noAuthClient := proto.NewStrongDocServiceClient(c.GetNoAuthConn())
	stream, err := noAuthClient.Login(context.Background())
	if err != nil {
		return err
	}
	defer stream.CloseSend()

	var userID string = ""
	var srpClient *srp.Client = nil
	_ = userID

	// Send the login prep message to server
	err = stream.Send(&proto.LoginReq{
		State: proto.LoginState_LOGIN_PREP,
		Data: &proto.LoginReq_PrepReq{
			PrepReq: &proto.PrepLoginReq{
				EmailOrUserID: userEmailOrID,
			},
		},
	})
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		switch resp.State {
		case proto.LoginState_LOGIN_PREP:
			_, srpClient, err = c.loginPrepResp(stream, resp, userEmailOrID, password)
			if err != nil {
				return err
			}
		case proto.LoginState_LOGIN_SRP_CRED:
			err = c.loginCredResp(stream, resp, srpClient)
			if err != nil {
				return err
			}
		case proto.LoginState_LOGIN_SRP_PROOF:
			err = c.loginProofResp(stream, resp, srpClient, password)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid login state %v", resp.State)
		}
	}
}

func (c *strongDocClientObj) loginPrepResp(stream proto.StrongDocService_LoginClient,
	resp *proto.LoginResp, userEmailOrID, password string) (string, *srp.Client, error) {

	lresp, ok := resp.GetData().(*proto.LoginResp_PrepResp)
	if !ok {
		return "", nil, fmt.Errorf("invalid login data type %v for state %v",
			reflect.TypeOf(resp.Data), resp.State)
	}

	userAuth := lresp.PrepResp.GetUserAuth()
	userID := lresp.PrepResp.GetUserID()
	if len(userID) == 0 {
		return "", nil, fmt.Errorf("can not find user for %v", userEmailOrID)
	}

	switch userAuth.GetAuthType() {
	case proto.AuthType_AUTH_SRP:
		srpAuth, ok := userAuth.GetAuth().(*proto.UserAuth_SrpAuth)
		if !ok {
			return "", nil, fmt.Errorf("invalid auth data type %v for auth %v user %v",
				reflect.TypeOf(userAuth.GetAuth()), userAuth.GetAuthType(),
				userID)
		}

		srpSession, err := srp.NewFromVersion(srpAuth.SrpAuth.GetSrpVersion())
		if err != nil {
			return "", nil, err
		}

		srpClient, err := srpSession.NewClient([]byte(userID), []byte(password))
		if err != nil {
			return "", nil, err
		}

		clientCred := srpClient.Credentials()

		fmt.Println("Client Cred: %v %v %v", userID, password, clientCred)

		err = stream.Send(&proto.LoginReq{
			State: proto.LoginState_LOGIN_SRP_CRED,
			Data: &proto.LoginReq_SrpCredReq{
				SrpCredReq: &proto.SrpCredReq{
					UserID: userID,
					ClientCreds: &proto.SrpCredential{
						SrpCredential: clientCred,
					},
				},
			},
		})
		if err != nil {
			return "", nil, err
		}

		return userID, srpClient, nil
	case proto.AuthType_AUTH_NONE:
		return "", nil, fmt.Errorf("unsupported user authentication type %v",
			lresp.PrepResp.GetUserAuth().GetAuthType())
	default:
		return "", nil, fmt.Errorf("unsupported user authentication type %v",
			lresp.PrepResp.GetUserAuth().GetAuthType())
	}
}

func (c *strongDocClientObj) loginCredResp(stream proto.StrongDocService_LoginClient,
	resp *proto.LoginResp, srpClient *srp.Client) error {
	lresp, ok := resp.GetData().(*proto.LoginResp_SrpCredResp)
	if !ok {
		return fmt.Errorf("invalid login data type %v for state %v",
			reflect.TypeOf(resp.Data), resp.State)
	}

	serverCred := lresp.SrpCredResp.GetServerCreds()
	clientProof, err := srpClient.Generate(serverCred.GetSrpCredential())
	if err != nil {
		return err
	}

	err = stream.Send(&proto.LoginReq{
		State: proto.LoginState_LOGIN_SRP_PROOF,
		Data: &proto.LoginReq_SrpProofReq{
			SrpProofReq: &proto.SrpProofReq{
				ClientProof: &proto.SrpAuthProof{
					SrpAuthProof: clientProof,
				},
			},
		},
	})

	return err
}

func (c *strongDocClientObj) loginProofResp(stream proto.StrongDocService_LoginClient,
	resp *proto.LoginResp, srpClient *srp.Client, password string) error {
	lresp, ok := resp.GetData().(*proto.LoginResp_SrpProofResp)
	if !ok {
		return fmt.Errorf("invalid login data type %v for state %v",
			reflect.TypeOf(resp.Data), resp.State)
	}

	if !lresp.SrpProofResp.GetClientSuccess() {
		return fmt.Errorf("login failed")
	}

	token := lresp.SrpProofResp.GetJtwToken()

	// Server proof is optional
	if len(lresp.SrpProofResp.GetServerProof().GetSrpAuthProof()) > 0 {
		ok := srpClient.ServerOk(lresp.SrpProofResp.GetServerProof().GetSrpAuthProof())
		if !ok {
			return fmt.Errorf("server failed to verify its identity")
		}
	}

	config := serviceLocations[c.location]

	// Close existing authenticated connection pool
	if c.authConnPool != nil {
		c.authConnPool.closeAll()
	}
	// Initialize connection pool
	connPool, err := initConnPool(MAX_CONCURRENT_CONNECTIONS,
		token, config.HostPort, config.Cert)
	if err != nil {
		return err
	}

	c.authConnPool = connPool
	c.authToken = token
	c.user, err = apidef.ConvertUserProtoToClient(lresp.SrpProofResp.GetUserData(), password)
	if err != nil {
		return err
	}

	return stream.CloseSend()
}

// func (c *strongDocClientObj) newAuthSession(prepareAuthResp *proto.PrepareAuthResp, authPurpose proto.AuthPurpose, userID, orgID, password string) (*AuthSession, error) {
// 	/*if authPurpose == proto.AuthPurpose_AUTH_LOGIN {
// 		prepareLoginResp, err := c.GetNoAuthConn().PrepareLogin(context.Background(), &proto.PrepareLoginReq{
// 			EmailOrUserID: userID,
// 			OrgID:         orgID,
// 		})
// 		if err != nil {
// 			err = fmt.Errorf("Login err: [%v]", err)
// 			return nil, err
// 		}

// 		prepareAuthResp = prepareLoginResp.GetPrepareAuthResp()
// 		realUserID = prepareLoginResp.GetUserID()

// 			switch prepareRes.GetAuthType() {
// 			case proto.AuthType_AUTH_SRP:
// 				return c.loginSRP(prepareRes.GetUserID(), password, orgID, prepareRes.GetAuthVersion())
// 			}
// 	} else {
// 		res, err := c.GetGrpcClient().PrepareAuth(context.Background(), &proto.PrepareAuthReq{})
// 		if err != nil {
// 			return nil, err
// 		}
// 		prepareAuthResp = res
// 	}*/

// 	switch prepareAuthResp.GetAuthType() {
// 	case proto.AuthType_AUTH_SRP:
// 		return c.authSRP(userID, password, orgID, prepareAuthResp.GetAuthVersion(), authPurpose)
// 	}

// 	return nil, fmt.Errorf("Unsupported authentication type: %v", prepareAuthResp.GetAuthType())
// }

// func (c *strongDocClientObj) NewAuthSession(password string) (*AuthSession, error) {
// 	res, err := c.GetGrpcClient().PrepareAuth(context.Background(), &proto.PrepareAuthReq{})
// 	if err != nil {
// 		return nil, err
// 	}
// 	return c.newAuthSession(res, proto.AuthPurpose_AUTH_PERSISTENT, c.userID, "", password)
// }

// func (auth *AuthSession) GetAuthID() string {
// 	return auth.authID
// }

// func (auth *AuthSession) CanAuthenticateData() bool {
// 	if auth.srpSharedKey == nil {
// 		if auth.srpClient == nil {
// 			return false
// 		}
// 		var err error
// 		auth.srpSharedKey, err = auth.srpClient.StrongSaltKey()
// 		if err != nil {
// 			return false
// 		}
// 	}
// 	return true
// }

// func (auth *AuthSession) PrepareDataForAuth(plaintext []byte) (string, error) {
// 	if auth.srpSharedKey == nil {
// 		if auth.srpClient == nil {
// 			return "", fmt.Errorf("This Authentication Session cannot authenticate data.")
// 		}
// 		var err error
// 		auth.srpSharedKey, err = auth.srpClient.StrongSaltKey()
// 		if err != nil {
// 			return "", fmt.Errorf("Error getting SRP shared key: %v", err)
// 		}
// 	}
// 	return auth.srpSharedKey.EncryptBase64(plaintext)
// }

// InitStrongDocClient initializes a singleton StrongDocClient
func InitStrongDocClient(location ServiceLocation, reset bool) (StrongDocClient, error) {
	_, ok := serviceLocations[location]
	if !ok || location == unset {
		return nil, fmt.Errorf("the ServiceLocation %v is not supported", location)
	}

	if atomic.LoadUint32(&clientInit) == 1 {
		if location == serviceLocation {
			return client, nil
		} else if !reset {
			return nil, fmt.Errorf("can not initialize StrongDocClient with service location %v. "+
				"Singleton already initialized with %v", location, serviceLocation)
		}
	}

	clientMutex.Lock()
	defer clientMutex.Unlock()

	if client != nil {
		client.Close()
	}

	var err error
	client, err = CreateStrongDocClient(location)
	if err != nil {
		return nil, err
	}

	serviceLocation = location
	atomic.StoreUint32(&clientInit, 1)
	return client, nil
}

// CreateStrongDocClient creates an instance of StrongDocClient
func CreateStrongDocClient(location ServiceLocation) (StrongDocClient, error) {
	config := serviceLocations[location]
	noAuthConn, err := getNoAuthConn(config.HostPort, config.Cert)
	if err != nil {
		return nil, err
	}
	client = &strongDocClientObj{
		location:   location,
		noAuthConn: noAuthConn,
	}
	return client, nil
}

// GetStrongDocClient gets a singleton StrongDocClient
func GetStrongDocClient() (StrongDocClient, error) {
	if atomic.LoadUint32(&clientInit) == 1 {
		if client != nil {
			return client, nil
		}
	}
	return nil, fmt.Errorf("can not get StrongDocManager. Please call InitStrongDocManager to initialize")
}

// GetStrongDocGrpcClient gets a singleton gRPC StrongDocServiceClient
func GetStrongDocGrpcClient() (proto.StrongDocServiceClient, error) {
	if atomic.LoadUint32(&clientInit) == 1 {
		if client != nil {
			return client.GetGrpcClient(), nil
		}
	}
	return nil, fmt.Errorf("can not get StrongDocClient. Please call InitStrongDocManager to initialize")
}

// func (c *strongDocClientObj) Logout() (status string, err error) {
// 	res, err := c.GetGrpcClient().Logout(context.Background(), &proto.LogoutReq{})
// 	if err != nil {
// 		return
// 	}
// 	status = res.Status

// 	c.passwordKey = nil
// 	c.passwordKeyID = ""

// 	return
// }

// func (c *strongDocClientObj) authSRP(userID, password, orgID string, version int32, authPurpose proto.AuthPurpose) (authSession *AuthSession, err error) {
// 	srpSession, err := srp.NewFromVersion(version)
// 	if err != nil {
// 		return nil, err
// 	}

// 	srpClient, err := srpSession.NewClient([]byte(userID), []byte(password))
// 	if err != nil {
// 		return nil, err
// 	}

// 	clientCreds := srpClient.Credentials()

// 	initRes, err := c.GetGrpcClient().SrpInit(context.Background(), &proto.SrpInitReq{
// 		AuthPurpose: authPurpose,
// 		UserID:      userID,
// 		OrgID:       orgID,
// 		ClientCreds: clientCreds,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	clientProof, err := srpClient.Generate(initRes.GetServerCreds())
// 	if err != nil {
// 		return nil, err
// 	}

// 	proofRes, err := c.GetGrpcClient().SrpProof(context.Background(), &proto.SrpProofReq{
// 		UserID:      userID,
// 		AuthID:      initRes.GetAuthID(),
// 		ClientProof: clientProof,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	ok := srpClient.ServerOk(proofRes.GetServerProof())
// 	if !ok {
// 		return nil, fmt.Errorf("Server failed to verify its identity.")
// 	}

// 	//loginRes := proofRes.GetLoginResponse()

// 	/*token := loginRes.GetToken()
// 	config := serviceLocations[c.location]
// 	authConn, err := getAuthConn(token, config.HostPort, config.Cert)
// 	if err != nil {
// 		return
// 	}

// 	// Close existing authenticated connection
// 	if c.authConn != nil {
// 		c.authConn.Close()
// 	}

// 	c.authConn = authConn
// 	c.authToken = loginRes.GetToken()

// 	encodedSerialKdfMeta := loginRes.GetKdfMeta()
// 	if encodedSerialKdfMeta != "" {
// 		serialKdfMeta, err := base64.URLEncoding.DecodeString(encodedSerialKdfMeta)
// 		if err != nil {
// 			return nil, err
// 		}
// 		userKdf, err := sscKdf.DeserializeKdf(serialKdfMeta)
// 		if err != nil {
// 			return nil, err
// 		}
// 		passwordKey, err := userKdf.GenerateKey([]byte(password))
// 		if err != nil {
// 			return nil, err
// 		}
// 		c.passwordKey = passwordKey
// 		c.passwordKeyID = loginRes.GetKeyID()
// 	}*/

// 	authSession = &AuthSession{
// 		authID:      initRes.GetAuthID(),
// 		authType:    proto.AuthType_AUTH_SRP,
// 		authVersion: version,
// 		loginResp:   proofRes.GetLoginResponse(),
// 		srpClient:   srpClient,
// 	}

// 	return
// }

// func (c *strongDocClientObj) ChangePassword(oldPassword, newPassword string) error {
// 	authSession, err := c.NewAuthSession(oldPassword)
// 	if err != nil {
// 		return err
// 	}

// 	setAuthReq := &proto.SetUserAuthMetadataReq{
// 		AuthID:         authSession.authID,
// 		NewAuthType:    authSession.authType,
// 		NewAuthVersion: authSession.authVersion,
// 	}

// 	switch authSession.authType {
// 	case proto.AuthType_AUTH_SRP:
// 		srpSession, err := srp.NewFromVersion(authSession.authVersion)
// 		if err != nil {
// 			return nil
// 		}
// 		newSrpVerifier, err := srpSession.Verifier([]byte(c.GetUserID()), []byte(newPassword))
// 		_, newVerifierString := newSrpVerifier.Encode()
// 		authSrpVerifierStr, err := authSession.PrepareDataForAuth([]byte(newVerifierString))
// 		if err != nil {
// 			return err
// 		}

// 		setAuthReq.SrpVerifier = authSrpVerifierStr
// 	default:
// 		return fmt.Errorf("Unsupported Authentication Type: %v", authSession.authType)
// 	}

// 	newKdf, err := sscKdf.New(sscKdf.Type_Argon2, ssc.Type_Secretbox)
// 	if err != nil {
// 		return nil
// 	}
// 	newKdfMetaBytes, err := newKdf.Serialize()
// 	if err != nil {
// 		return nil
// 	}
// 	newPasswordKey, err := newKdf.GenerateKey([]byte(newPassword))
// 	if err != nil {
// 		return nil
// 	}

// 	authKdfMetaStr, err := authSession.PrepareDataForAuth(newKdfMetaBytes)
// 	if err != nil {
// 		return err
// 	}

// 	setAuthReq.KdfMeta = authKdfMetaStr

// 	done := false
// 	attempts := 0
// 	maxAttempts := 5

// 	var setAuthResp *proto.SetUserAuthMetadataResp

// 	for !done {
// 		if attempts >= maxAttempts {
// 			return fmt.Errorf("ChangePassword Error: max attempts exceeded.")
// 		}

// 		attempts += 1

// 		keysReq := &proto.GetUserPrivateKeysReq{}
// 		keysResp, err := c.GetGrpcClient().GetUserPrivateKeys(context.Background(), keysReq)
// 		if err != nil {
// 			return err
// 		}

// 		oldEncKeys := keysResp.GetEncryptedKeys()
// 		newEncKeys := make([]*proto.EncryptedKey, len(oldEncKeys))
// 		for i, oldEncKey := range oldEncKeys {
// 			if oldEncKey.EncryptorID != c.GetUserKeyID() {
// 				return fmt.Errorf("ChangePassword Error: User information out of date. User must log out and log back in again before continuing.")
// 			}
// 			keyBytes, err := c.UserDecryptBase64(oldEncKey.EncKey)
// 			if err != nil {
// 				return err
// 			}
// 			newEncKeyBytes, err := newPasswordKey.Encrypt(keyBytes)
// 			if err != nil {
// 				return err
// 			}
// 			authEncKeyStr, err := authSession.PrepareDataForAuth(newEncKeyBytes)
// 			if err != nil {
// 				return err
// 			}
// 			newEncKeys[i] = &proto.EncryptedKey{
// 				EncKey:     authEncKeyStr,
// 				KeyID:      oldEncKey.GetKeyID(),
// 				KeyVersion: oldEncKey.GetKeyVersion(),
// 				OwnerID:    oldEncKey.GetOwnerID(),
// 				OwnerType:  oldEncKey.GetOwnerType(),
// 			}
// 		}

// 		setAuthReq.EncryptedKeys = newEncKeys

// 		setAuthResp, err = c.GetGrpcClient().SetUserAuthMetadata(context.Background(), setAuthReq)
// 		if err != nil {
// 			return err
// 		}

// 		done = !setAuthResp.GetRestart()
// 	}

// 	// Get KeyID and save stuff in client
// 	// Wait, actually force logout

// 	return nil
// }

// GetNoAuthConn get the unauthenticated GRPC connection. This is always available, but will not work in most API calls
func (c *strongDocClientObj) GetNoAuthConn() *grpc.ClientConn {
	return c.noAuthConn
}

// GetAuthConn gets authenticated GRPC connection pool. This is available after a successful login.
func (c *strongDocClientObj) GetAuthConnPool() *connPool {
	return c.authConnPool
}

// GetProtoClient returns a gRPC StrongDocServiceClient used to call GRPC functions
func (c *strongDocClientObj) GetGrpcClient() proto.StrongDocServiceClient {
	authConnPool := c.GetAuthConnPool()
	if authConnPool != nil {
		return proto.NewStrongDocServiceClient(authConnPool.getConn())
	}
	return proto.NewStrongDocServiceClient(c.GetNoAuthConn())
}

type GrpcClient struct {
	Client proto.StrongDocServiceClient
	Error  error
}

// Close closes all the connections.
func (c *strongDocClientObj) Close() {
	if c.GetNoAuthConn() != nil {
		c.GetNoAuthConn().Close()
	}
	if c.GetAuthConnPool() != nil {
		c.GetAuthConnPool().closeAll()
	}
}

/*func (c *strongDocClientObj) GetPasswordKey() *ssc.StrongSaltKey {
	return c.passwordKey
}*/

// func (c *strongDocClientObj) UserEncrypt(plaintext []byte) ([]byte, error) {
// 	return c.passwordKey.Encrypt(plaintext)
// }

// func (c *strongDocClientObj) UserEncryptBase64(plaintext []byte) (string, error) {
// 	return c.passwordKey.EncryptBase64(plaintext)
// }
// func (c *strongDocClientObj) UserDecrypt(ciphertext []byte) ([]byte, error) {
// 	return c.passwordKey.Decrypt(ciphertext)
// }

// func (c *strongDocClientObj) UserDecryptBase64(ciphertext string) ([]byte, error) {
// 	return c.passwordKey.DecryptBase64(ciphertext)
// }

// func (c *strongDocClientObj) GetUserID() string {
// 	return c.userID
// }

// func (c *strongDocClientObj) GetUserKeyID() string {
// 	return c.passwordKeyID
// }

func getNoAuthConn(hostport, cert string) (conn *grpc.ClientConn, err error) {
	certFilePath, err := utils.FetchFileLoc(cert)

	// Create the client TLS credentials
	creds, err := credentials.NewClientTLSFromFile(certFilePath, "")
	if err != nil {
		err = fmt.Errorf("Can not load TLS cert at %v", cert)
		return
	}

	// Initiate a connection with the server
	return grpc.DialContext(context.Background(), hostport,
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(&grpcNoAuthCred{}),
	)
}

func getAuthConn(token, hostport, cert string) (conn *grpc.ClientConn, err error) {
	certFilePath, err := utils.FetchFileLoc(cert)

	// Create the client TLS credentials
	creds, err := credentials.NewClientTLSFromFile(certFilePath, "")
	if err != nil {
		log.Fatalf("could not load tls cert: %s", err)
		return
	}

	// Initiate a connection with the server
	return grpc.DialContext(context.Background(), hostport,
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(&grpcAuthCred{token}),
	)
}
