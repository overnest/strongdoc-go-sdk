package utils

//type DocIndexType int

type OwnerType string

const (
	// seek whence
	SeekSet = 0
	SeekCur = 1
	SeekEnd = 2

	// max receive msg limit
	onegb          = 1024 * 1024 * 1024
	MaxRecvMsgSize = onegb*2 + 1000 // 9223372036854775807 max int (int64)  value

	// Owner type
	OwnerOrg  OwnerType = "OWNER_ORG"
	OwnerUser OwnerType = "OWNER_USER"

	TestLocal bool = false
)
