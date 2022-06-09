package apidef

import (
	"fmt"
	"time"

	"github.com/overnest/strongdoc-go-sdk/proto"
	"github.com/overnest/strongsalt-crypto-go/pake/srp"
)

const (
	SRP_VER = int32(1)
)

type UserAuth struct {
	AuthType proto.AuthType
	SrpAuth  *SrpAuth
}

type SrpAuth struct {
	SrpVersion  int32
	SrpVerifier *srp.Verifier
}

type UserCred struct {
	KdfKey   *Key
	AsymKey  *Key
	AsymKeys []*Key
}

type SearchCred struct {
	TermKey    *Key
	TermKeys   []*Key
	SearchKey  *Key
	SearchKeys []*Key
}

type User struct {
	UserID     string
	Password   string // This can only be provided by client
	Name       string
	Email      string
	UserAuth   *UserAuth
	UserCred   *UserCred
	SearchCred *SearchCred
	CreatedAt  time.Time
}

func ConvertUserProtoToClient(puser *proto.User, password string) (*User, error) {
	if puser == nil {
		return nil, nil
	}

	user := &User{
		UserID:     puser.GetUserID(),
		Password:   password,
		Email:      puser.GetEmail(),
		Name:       puser.GetName(),
		CreatedAt:  puser.GetCreatedAt().AsTime(),
		UserAuth:   nil,
		UserCred:   nil,
		SearchCred: nil,
	}

	// Convert User Authentication
	user.UserAuth = &UserAuth{
		AuthType: puser.GetUserAuth().AuthType,
		SrpAuth:  nil,
	}

	switch puser.GetUserAuth().AuthType {
	case proto.AuthType_AUTH_SRP:
		if srpAuth, ok := puser.GetUserAuth().GetAuth().(*proto.UserAuth_SrpAuth); ok {
			srpSession, err := srp.NewFromVersion(srpAuth.SrpAuth.GetSrpVersion())
			if err != nil {
				return nil, err
			}

			srpVerifier, err := srpSession.Verifier([]byte(puser.GetUserID()), []byte(password))
			if err != nil {
				return nil, err
			}

			user.UserAuth.SrpAuth = &SrpAuth{
				SrpVersion:  srpAuth.SrpAuth.GetSrpVersion(),
				SrpVerifier: srpVerifier,
			}
		}
	case proto.AuthType_AUTH_NONE:
		// Do nothing
	default:
		return nil, fmt.Errorf("user authentication type %v is not supported",
			puser.GetUserAuth().AuthType)
	}

	// Convert User Credentials
	puserCred := puser.GetUserCred()
	if puserCred != nil {
		kdfKey, err := ParseKdfProtoKey(puserCred.GetKdfKey(), password)
		if err != nil {
			return nil, err
		}

		asymKey, err := ParseProtoKey(puserCred.GetAsymKey())
		if err != nil {
			return nil, err
		}

		user.UserCred = &UserCred{
			KdfKey:   kdfKey,
			AsymKey:  asymKey,
			AsymKeys: make([]*Key, 0, len(puserCred.AsymKeys)),
		}

		for _, pak := range puserCred.AsymKeys {
			ak, err := ParseProtoKey(pak, kdfKey)
			if err != nil {
				return nil, err
			}
			user.UserCred.AsymKeys = append(user.UserCred.AsymKeys, ak)
		}
	}

	// Convert Search Credentials
	psearchCred := puser.GetSearchCred()
	if psearchCred != nil {
		termKey, err := ParseProtoKey(psearchCred.GetTermKey())
		if err != nil {
			return nil, err
		}

		searchKey, err := ParseProtoKey(psearchCred.GetSearchKey())
		if err != nil {
			return nil, err
		}

		user.SearchCred = &SearchCred{
			TermKey:    termKey,
			SearchKey:  searchKey,
			TermKeys:   make([]*Key, 0, len(psearchCred.GetTermKeys())),
			SearchKeys: make([]*Key, 0, len(psearchCred.GetSearchKeys())),
		}

		for _, ptk := range psearchCred.GetTermKeys() {
			tk, err := ParseProtoKey(ptk, user.UserCred.KdfKey)
			if err != nil {
				return nil, err
			}
			user.SearchCred.TermKeys = append(user.SearchCred.TermKeys, tk)
		}

		for _, psk := range psearchCred.GetSearchKeys() {
			sk, err := ParseProtoKey(psk, user.UserCred.KdfKey)
			if err != nil {
				return nil, err
			}
			user.SearchCred.SearchKeys = append(user.SearchCred.SearchKeys, sk)
		}
	}

	return user, nil
}
