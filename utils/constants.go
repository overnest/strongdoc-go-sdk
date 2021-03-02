package utils

type DocIndexType int

type OwnerType int

const (
	// document index type
	OffsetIndex DocIndexType = 0
	TermIndex   DocIndexType = 1

	// seek whence
	SeekSet = 0
	SeekCur = 1
	SeekEnd = 2

	// max receive msg limit
	onegb          = 1024 * 1024 * 1024
	MaxRecvMsgSize = onegb*2 + 1000 // 9223372036854775807 max int (int64)  value

	// Owner type
	Owner_Org  OwnerType = 1
	Owner_User OwnerType = 2
)
