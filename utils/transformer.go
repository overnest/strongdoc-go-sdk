package utils

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/proto"
)

func TranslateOwnerType(ownerType OwnerType) (accessType proto.AccessType, err error) {
	switch ownerType {
	case OwnerUser:
		accessType = proto.AccessType_USER
		break
	case OwnerOrg:
		accessType = proto.AccessType_ORG
		break
	default:
		err = fmt.Errorf("unsuppported ownerType")
	}
	return
}
