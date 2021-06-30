package searchidxv2

//////////////////////////////////////////////////////////////////
//
//                  Search Term Index Block
//
//////////////////////////////////////////////////////////////////

// SearchTermIdxBlkV1 is the Search Index Block V1
type SearchTermIdxBlkV2 struct {
	TermDocVerOffset  map[string](map[string]VersionOffsetV2) // term -> (docID -> versionOffsets)
	predictedJSONSize uint64                                  `json:"-"`
	maxDataSize       uint64                                  `json:"-"`
}

// VersionOffsetV2 stores the document version and associated offsets
type VersionOffsetV2 struct {
	Version uint64
	Offsets []uint64
}
