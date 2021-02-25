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
