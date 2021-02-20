package searchidx

// "io"
// "github.com/go-errors/errors"
// ssheaders "github.com/overnest/strongsalt-common-go/headers"

// sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"

import (
	"io"
	"strings"

	"github.com/emirpasic/gods/trees/binaryheap"
	"github.com/overnest/strongdoc-go-sdk/utils"

	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

var batchSourceComparator func(a, b interface{}) int = func(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	wa := a.(*SearchTermIdxWriterV1)
	wb := b.(*SearchTermIdxWriterV1)
	return strings.Compare(wa.Term, wb.Term)
}

//////////////////////////////////////////////////////////////////
//
//                   Search Term Batch Manager V1
//
//////////////////////////////////////////////////////////////////

type SearchTermBatchMgrV1 struct {
	SearchIdxSources []SearchIdxSourceV1
	TermKey          *sscrypto.StrongSaltKey
	IndexKey         *sscrypto.StrongSaltKey
	delDocs          *DeletedDocsV1
	termHeap         *binaryheap.Heap
}

func CreateSearchTermBatchMgrV1(sources []SearchIdxSourceV1,
	termKey, indexKey *sscrypto.StrongSaltKey) (*SearchTermBatchMgrV1, error) {

	mgr := &SearchTermBatchMgrV1{
		SearchIdxSources: sources,
		TermKey:          termKey,
		IndexKey:         indexKey,
		delDocs:          nil,
		termHeap:         binaryheap.NewWith(batchSourceComparator)}

	termWriterMap := make(map[string]*SearchTermIdxWriterV1)
	for _, source := range sources {
		for _, term := range source.GetAddTerms() {
			stiw := termWriterMap[term]
			if stiw == nil {
				termWriterMap[term] = CreateSearchTermIdxWriterV1(term, nil, nil, termKey, indexKey)
				mgr.termHeap.Push(stiw)
			}
			stiw.AddSource = append(stiw.AddSource, source)
		}

		for _, term := range source.GetDelTerms() {
			stiw := termWriterMap[term]
			if stiw == nil {
				termWriterMap[term] = CreateSearchTermIdxWriterV1(term, nil, nil, termKey, indexKey)
				mgr.termHeap.Push(stiw)
			}
			stiw.DelSource = append(stiw.DelSource, source)
		}
		// stbs := &searchTermBatchSourceV1{source, source.GetAddTerms(), source.GetDelTerms()}
		// mgr.batchSources = append(mgr.batchSources, stbs)
	}

	return mgr, nil
}

func (mgr *SearchTermBatchMgrV1) GetNextTermBatch(batchSize int) (*SearchTermBatchV1, error) {
	batch := &SearchTermBatchV1{
		SourceToWriters: make(map[SearchIdxSourceV1][]*SearchTermIdxWriterV1),
		SourceList:      nil,
		TermToWriter:    make(map[string]*SearchTermIdxWriterV1),
	}

	for i := 0; i < batchSize; i++ {
		s, _ := mgr.termHeap.Pop()
		if s == nil {
			break
		}

		stiw := s.(*SearchTermIdxWriterV1)
		batch.TermToWriter[stiw.Term] = stiw
	}

	sourceMap := make(map[SearchIdxSourceV1]map[*SearchTermIdxWriterV1]bool)
	for _, stiw := range batch.TermToWriter {
		for _, source := range stiw.AddSource {
			stbiMap := sourceMap[source]
			if stbiMap == nil {
				stbiMap = make(map[*SearchTermIdxWriterV1]bool)
				sourceMap[source] = stbiMap
			}
			if !stbiMap[stiw] {
				stbiMap[stiw] = true
			}
		}
	}

	for sis, stbiMap := range sourceMap {
		stbiList := make([]*SearchTermIdxWriterV1, 0, len(stbiMap))
		for stbi := range stbiMap {
			stbiList = append(stbiList, stbi)
		}
		batch.SourceToWriters[sis] = stbiList
	}

	batch.SourceList = make([]SearchIdxSourceV1, 0, len(batch.SourceToWriters))
	for sis := range batch.SourceToWriters {
		batch.SourceList = append(batch.SourceList, sis)
	}

	return batch, nil
}

//////////////////////////////////////////////////////////////////
//
//                    Search Term Batch V1
//
//////////////////////////////////////////////////////////////////

type SearchTermBatchV1 struct {
	SourceToWriters map[SearchIdxSourceV1][]*SearchTermIdxWriterV1
	SourceList      []SearchIdxSourceV1
	TermToWriter    map[string]*SearchTermIdxWriterV1 // term -> SearchTermIdxWriterV1
}

func (batch *SearchTermBatchV1) ProcessBatch() (map[*SearchTermIdxWriterV1]*SearchTermIdxWriterRespV1, error) {

	writerToBlocks := make(map[*SearchTermIdxWriterV1][]*SearchIdxSourceBlockV1)

	for _, source := range batch.SourceList {
		block, err := source.GetNextSourceBlock(nil)
		if err != nil {
			if err == io.EOF {
				continue
			}
			return nil, err
		}
		for _, stiw := range batch.SourceToWriters[source] {
			sisbList := writerToBlocks[stiw]
			if sisbList == nil {
				sisbList = make([]*SearchIdxSourceBlockV1, 0, len(stiw.AddSource))
			}
			writerToBlocks[stiw] = append(sisbList, block)
		}
	}

	writerToChan := make(map[*SearchTermIdxWriterV1](chan *SearchTermIdxWriterRespV1))
	for stiw, sisbList := range writerToBlocks {
		stiwChan := make(chan *SearchTermIdxWriterRespV1)
		writerToChan[stiw] = stiwChan
		go func(stiw *SearchTermIdxWriterV1, sisbList []*SearchIdxSourceBlockV1,
			stiwChan chan<- *SearchTermIdxWriterRespV1) {

			stibs, err := stiw.ProcessSourceBlocks(sisbList)
			stiwChan <- &SearchTermIdxWriterRespV1{stibs, err}
			if err != nil {
				close(stiwChan)
			}
		}(stiw, sisbList, stiwChan)
	}

	respMap := make(map[*SearchTermIdxWriterV1]*SearchTermIdxWriterRespV1)
	for writer, stiwChan := range writerToChan {
		respMap[writer] = <-stiwChan
	}

	return respMap, nil
}

//////////////////////////////////////////////////////////////////
//
//                  Search Term Index Writer V1
//
//////////////////////////////////////////////////////////////////

type SearchTermIdxWriterRespV1 struct {
	Blocks []*SearchTermIdxBlkV1
	Error  error
}

type SearchTermIdxWriterV1 struct {
	Term          string
	AddSource     []SearchIdxSourceV1
	DelSource     []SearchIdxSourceV1
	TermKey       *sscrypto.StrongSaltKey
	IndexKey      *sscrypto.StrongSaltKey
	delDocMap     map[string]bool   // DocID -> boolean
	highDocVerMap map[string]uint64 // DocID -> DocVer
	newSti        *SearchTermIdxV1
	oldSti        SearchTermIdx
	newStiBlock   *SearchTermIdxBlkV1
	oldStiBlock   *SearchTermIdxBlkV1
	oldStiDocIDs  []string
}

func CreateSearchTermIdxWriterV1(term string, addSource, delSource []SearchIdxSourceV1,
	termKey, indexKey *sscrypto.StrongSaltKey) *SearchTermIdxWriterV1 {

	if addSource == nil {
		addSource = make([]SearchIdxSourceV1, 0, 10)
	}
	if delSource == nil {
		delSource = make([]SearchIdxSourceV1, 0, 10)
	}
	return &SearchTermIdxWriterV1{
		Term:          term,
		AddSource:     addSource,
		DelSource:     delSource,
		TermKey:       termKey,
		IndexKey:      indexKey,
		delDocMap:     make(map[string]bool),
		highDocVerMap: make(map[string]uint64),
		newSti:        nil,
		oldSti:        nil,
		newStiBlock:   nil,
		oldStiBlock:   nil,
		oldStiDocIDs:  make([]string, 0),
	}
}

func (stiw *SearchTermIdxWriterV1) ProcessSourceBlocks(sourceBlocks []*SearchIdxSourceBlockV1) ([]*SearchTermIdxBlkV1, error) {
	var err error
	copyLen := 10

	if stiw.newSti == nil {
		owner := CreateSearchIdxOwner(SI_OWNER_USR, "USER1")
		stiw.newSti, err = CreateSearchTermIdxV1(owner, stiw.Term, stiw.TermKey, stiw.IndexKey, nil, nil)
		if err != nil {
			return nil, err
		}

		for _, ds := range stiw.DelSource {
			stiw.delDocMap[ds.GetDocID()] = true
		}

		// If any of the document versions are newer than the one in the old search index,
		// they need to be deleted
		// if oldSortedDocIdx != nil {
		// 	// Comparison is much easier if there is sorted doc index

		// } else if oldSearchIdx != nil {
		// 	// If there isn't sorted doc index, then it can also be done with old search index
		// }
	}

	// Find the highest version of each DocID
	// TODO: account for the old search index
	for _, sisb := range sourceBlocks {
		if ver, exist := stiw.highDocVerMap[sisb.DocID]; !exist || ver < sisb.DocVer {
			stiw.highDocVerMap[sisb.DocID] = sisb.DocVer
		}
	}

	// Remove all sources with lower version than highest document version
	for i := 0; i < len(sourceBlocks); i++ {
		blk := sourceBlocks[i]
		if ver, exist := stiw.highDocVerMap[blk.DocID]; exist && ver > blk.DocVer {
			sourceBlocks = removeSourceBlock(sourceBlocks, i)
			i--
		}
	}

	// Read the old STI block
	if stiw.oldStiBlock == nil {
		err = stiw.getOldSearchTermIndexBlock()
		if err != nil && err != io.EOF {
			return nil, err
		}
	}

	// Create the new STI block to be written to
	if stiw.newStiBlock == nil {
		stiw.newStiBlock = CreateSearchTermIdxBlkV1(stiw.newSti.GetMaxBlockDataSize())
	}

	srcOffsetMap := make(map[*SearchIdxSourceBlockV1][]uint64)
	for _, srcBlk := range sourceBlocks {
		if _, ok := srcBlk.TermOffset[stiw.Term]; ok {
			// Clone the offset array
			srcOffsetMap[srcBlk] = append([]uint64{}, srcBlk.TermOffset[stiw.Term]...)
		}
	}

	// Process incoming source blocks until all are empty
	for sourceNotEmpty := true; sourceNotEmpty; {
		sourceNotEmpty = false

		// Process the source blocks
		for _, srcBlk := range sourceBlocks {
			srcOffsets := srcOffsetMap[srcBlk]
			if srcOffsets != nil && len(srcOffsets) > 0 {
				sourceNotEmpty = true
				count := utils.Min(len(srcOffsets), copyLen)
				offsets := srcOffsets[:count]
				err = stiw.newStiBlock.AddDocOffsets(srcBlk.DocID, srcBlk.DocVer, offsets)

				// Current block is full. Send it
				if err != nil {
					err = stiw.newSti.WriteBlock(stiw.newStiBlock)
					if err != nil {
						return nil, err
					}

					stiw.newStiBlock = CreateSearchTermIdxBlkV1(stiw.newSti.GetMaxBlockDataSize())
					err = stiw.newStiBlock.AddDocOffsets(srcBlk.DocID, srcBlk.DocVer, offsets)
					if err != nil {
						return nil, err
					}
				}

				srcOffsetMap[srcBlk] = srcOffsets[count:]
			}
		}

		if stiw.oldStiBlock != nil {
			for i := 0; i < len(stiw.oldStiDocIDs); i++ {
				docID := stiw.oldStiDocIDs[i]
				verOffset := stiw.oldStiBlock.DocVerOffset[docID]
				if verOffset == nil {
					stiw.oldStiDocIDs = removeString(stiw.oldStiDocIDs, i)
					i--
					continue
				}

				count := utils.Min(len(verOffset.Offsets), copyLen)
				offsets := verOffset.Offsets[:count]
				err = stiw.newStiBlock.AddDocOffsets(docID, verOffset.Version, offsets)
				if err != nil {
					// The new block is full. Don't bother processing anymore old STI block
					break
				}

				// Remove the successfully processed offsets from the old STI block
				verOffset.Offsets = verOffset.Offsets[count:]
				if len(verOffset.Offsets) == 0 {
					stiw.oldStiDocIDs = removeString(stiw.oldStiDocIDs, i)
					i--
				}
			} // for i := 0; i < len(stiw.oldStiDocIDs); i++

			if len(stiw.oldStiDocIDs) == 0 {
				err = stiw.getOldSearchTermIndexBlock()
				if err != nil && err != io.EOF {
					return nil, err
				}
			}
		} // if stiw.oldStiBlock != nil
	} // for sourceNotEmpty := true; sourceNotEmpty;

	return nil, nil
}

func (stiw *SearchTermIdxWriterV1) getOldSearchTermIndexBlock() error {
	if stiw.oldSti != nil {
		// TODO:
		// 1. Remove all document versions lower than high version
		// 2. Remove all documents in source block delete list
		// 3. Remove all documents in global delete list
		// 4. Set stiw.oldStiBlock
		// 5. Set stiw.oldStiDocIDs
	}

	return nil
}

func removeSourceBlock(blocks []*SearchIdxSourceBlockV1, i int) []*SearchIdxSourceBlockV1 {
	if blocks != nil && i >= 0 && i < len(blocks) {
		blocks[i] = blocks[len(blocks)-1]
		blocks = blocks[:len(blocks)-1]
	}
	return blocks
}

func removeString(stringList []string, i int) []string {
	if stringList != nil && i >= 0 && i < len(stringList) {
		stringList[i] = stringList[len(stringList)-1]
		stringList = stringList[:len(stringList)-1]
	}
	return stringList
}
