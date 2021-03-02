package utils

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/proto"
)

func TranslateDocIndexType(indexType DocIndexType) (docIndexType proto.DocIndexType, err error) {
	switch indexType {
	case OffsetIndex:
		docIndexType = proto.DocIndexType_OFFSET_INDEX
		break
	case TermIndex:
		docIndexType = proto.DocIndexType_TERM_INDEX
		break
	default:
		err = fmt.Errorf("unsuppported docIndexType")
	}
	return
}

func TranslateOwnerType(ownerType OwnerType) (accessType proto.AccessType, err error) {
	switch ownerType {
	case Owner_User:
		accessType = proto.AccessType_USER
		break
	case Owner_Org:
		accessType = proto.AccessType_ORG
		break
	default:
		err = fmt.Errorf("unsuppported ownerType")
	}
	return
}
