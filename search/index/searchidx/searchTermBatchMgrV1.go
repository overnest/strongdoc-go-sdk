package searchidx

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
	wa := a.(*SearchTermBatchWriterV1)
	wb := b.(*SearchTermBatchWriterV1)
	return strings.Compare(wa.Term, wb.Term)
}

var copyLen int = 10

//////////////////////////////////////////////////////////////////
//
//                   Search Term Batch Manager V1
//
//////////////////////////////////////////////////////////////////

type SearchTermBatchMgrV1 struct {
	Owner            SearchIdxOwner
	SearchIdxSources []SearchTermIdxSourceV1
	TermKey          *sscrypto.StrongSaltKey
	IndexKey         *sscrypto.StrongSaltKey
	delDocs          *DeletedDocsV1
	termHeap         *binaryheap.Heap
}

func CreateSearchTermBatchMgrV1(owner SearchIdxOwner, sources []SearchTermIdxSourceV1,
	termKey, indexKey *sscrypto.StrongSaltKey, delDocs *DeletedDocsV1) (*SearchTermBatchMgrV1, error) {

	var err error

	mgr := &SearchTermBatchMgrV1{
		Owner:    owner,
		TermKey:  termKey,
		IndexKey: indexKey,
		delDocs:  delDocs,
		termHeap: binaryheap.NewWith(batchSourceComparator)}

	termWriterMap := make(map[string]*SearchTermBatchWriterV1)
	for _, source := range sources {
		for _, term := range source.GetAddTerms() {
			stbw := termWriterMap[term]
			if stbw == nil {
				termWriterMap[term], err = CreateSearchTermBatchWriterV1(owner, term,
					nil, nil, termKey, indexKey, delDocs)
				if err != nil {
					return nil, err
				}
				mgr.termHeap.Push(stbw)
			}
			stbw.AddSource = append(stbw.AddSource, source)
		}

		for _, term := range source.GetDelTerms() {
			stbw := termWriterMap[term]
			if stbw == nil {
				termWriterMap[term], err = CreateSearchTermBatchWriterV1(owner, term,
					nil, nil, termKey, indexKey, delDocs)
				if err != nil {
					return nil, err
				}
				mgr.termHeap.Push(stbw)
			}
			stbw.DelSource = append(stbw.DelSource, source)
		}
	}

	return mgr, nil
}

func (mgr *SearchTermBatchMgrV1) GetNextTermBatch(batchSize int) (*SearchTermBatchV1, error) {
	batch := &SearchTermBatchV1{
		Owner:           mgr.Owner,
		SourceToWriters: make(map[SearchTermIdxSourceV1][]*SearchTermBatchWriterV1),
		SourceList:      nil,
		TermToWriter:    make(map[string]*SearchTermBatchWriterV1),
	}

	for i := 0; i < batchSize; i++ {
		s, _ := mgr.termHeap.Pop()
		if s == nil {
			break
		}

		stbw := s.(*SearchTermBatchWriterV1)
		batch.TermToWriter[stbw.Term] = stbw
	}

	sourceMap := make(map[SearchTermIdxSourceV1]map[*SearchTermBatchWriterV1]bool)
	for _, stbw := range batch.TermToWriter {
		for _, source := range stbw.AddSource {
			stbiMap := sourceMap[source]
			if stbiMap == nil {
				stbiMap = make(map[*SearchTermBatchWriterV1]bool)
				sourceMap[source] = stbiMap
			}
			if !stbiMap[stbw] {
				stbiMap[stbw] = true
			}
		}
	}

	for sis, stbiMap := range sourceMap {
		stbiList := make([]*SearchTermBatchWriterV1, 0, len(stbiMap))
		for stbi := range stbiMap {
			stbiList = append(stbiList, stbi)
		}
		batch.SourceToWriters[sis] = stbiList
	}

	batch.SourceList = make([]SearchTermIdxSourceV1, 0, len(batch.SourceToWriters))
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
	Owner           SearchIdxOwner
	SourceToWriters map[SearchTermIdxSourceV1][]*SearchTermBatchWriterV1
	SourceList      []SearchTermIdxSourceV1
	TermToWriter    map[string]*SearchTermBatchWriterV1 // term -> SearchTermIdxWriterV1
}

func (batch *SearchTermBatchV1) ProcessBatch() (map[*SearchTermBatchWriterV1]*SearchTermBatchWriterRespV1, error) {

	writerToBlocks := make(map[*SearchTermBatchWriterV1][]*SearchTermIdxSourceBlockV1)

	for _, source := range batch.SourceList {
		block, err := source.GetNextSourceBlock(nil)
		if err != nil {
			if err == io.EOF {
				continue
			}
			return nil, err
		}
		for _, stbw := range batch.SourceToWriters[source] {
			sisbList := writerToBlocks[stbw]
			if sisbList == nil {
				sisbList = make([]*SearchTermIdxSourceBlockV1, 0, len(stbw.AddSource))
			}
			writerToBlocks[stbw] = append(sisbList, block)
		}
	}

	writerToChan := make(map[*SearchTermBatchWriterV1](chan *SearchTermBatchWriterRespV1))
	for stbw, sisbList := range writerToBlocks {
		stbwChan := make(chan *SearchTermBatchWriterRespV1)
		writerToChan[stbw] = stbwChan
		go func(stbw *SearchTermBatchWriterV1, sisbList []*SearchTermIdxSourceBlockV1,
			stbwChan chan<- *SearchTermBatchWriterRespV1) {

			stibs, err := stbw.ProcessSourceBlocks(sisbList)
			stbwChan <- &SearchTermBatchWriterRespV1{stibs, err}
			if err != nil {
				close(stbwChan)
			}
		}(stbw, sisbList, stbwChan)
	}

	respMap := make(map[*SearchTermBatchWriterV1]*SearchTermBatchWriterRespV1)
	for writer, stbwChan := range writerToChan {
		respMap[writer] = <-stbwChan
	}

	return respMap, nil
}

//////////////////////////////////////////////////////////////////
//
//                 Search Term Batch Writer V1
//
//////////////////////////////////////////////////////////////////

type SearchTermBatchWriterRespV1 struct {
	Blocks []*SearchTermIdxBlkV1
	Error  error
}

type SearchTermBatchWriterV1 struct {
	Term            string
	Owner           SearchIdxOwner
	AddSource       []SearchTermIdxSourceV1
	DelSource       []SearchTermIdxSourceV1
	TermKey         *sscrypto.StrongSaltKey
	IndexKey        *sscrypto.StrongSaltKey
	delDocMap       map[string]bool   // DocID -> boolean
	highDocVerMap   map[string]uint64 // DocID -> DocVer
	newSti          *SearchTermIdxV1
	oldSti          SearchTermIdx
	newStiBlk       *SearchTermIdxBlkV1
	oldStiBlk       *SearchTermIdxBlkV1
	oldStiBlkDocIDs []string
}

func CreateSearchTermBatchWriterV1(owner SearchIdxOwner, term string,
	addSource, delSource []SearchTermIdxSourceV1,
	termKey, indexKey *sscrypto.StrongSaltKey, delDocs *DeletedDocsV1) (*SearchTermBatchWriterV1, error) {

	var err error

	if addSource == nil {
		addSource = make([]SearchTermIdxSourceV1, 0, 10)
	}
	if delSource == nil {
		delSource = make([]SearchTermIdxSourceV1, 0, 10)
	}

	stbw := &SearchTermBatchWriterV1{
		Owner:           owner,
		Term:            term,
		AddSource:       addSource,
		DelSource:       delSource,
		TermKey:         termKey,
		IndexKey:        indexKey,
		delDocMap:       make(map[string]bool),
		highDocVerMap:   make(map[string]uint64),
		newSti:          nil,
		oldSti:          nil,
		newStiBlk:       nil,
		oldStiBlk:       nil,
		oldStiBlkDocIDs: make([]string, 0),
	}

	// Record the DocIDs to be deleted
	for _, ds := range delSource {
		stbw.delDocMap[ds.GetDocID()] = true
	}

	if delDocs != nil {
		for _, docID := range delDocs.DelDocs {
			stbw.delDocMap[docID] = true
		}
	}

	// Find the highest version of each DocID. This is so we can remove
	// any DocID with lower version
	for _, stis := range addSource {
		if ver, exist := stbw.highDocVerMap[stis.GetDocID()]; !exist || ver < stis.GetDocVer() {
			stbw.highDocVerMap[stis.GetDocID()] = stis.GetDocVer()
		}
	}
	for _, stis := range delSource {
		if ver, exist := stbw.highDocVerMap[stis.GetDocID()]; !exist || ver < stis.GetDocVer() {
			stbw.highDocVerMap[stis.GetDocID()] = stis.GetDocVer()
		}
	}

	updateID, err := GetLatestUpdateIdV1(owner, term, termKey)
	if err != nil {
		return nil, err
	}

	if updateID != "" {
		stbw.oldSti, err = OpenSearchTermIdxV1(owner, term, termKey, indexKey, updateID)
		if err != nil {
			stbw.oldSti = nil
		}
		err = stbw.updateHighDocVersion(owner, term, termKey, indexKey, updateID)
		if err != nil {
			return nil, err
		}
	}

	stbw.newSti, err = CreateSearchTermIdxV1(owner, term, termKey, indexKey, nil, nil)
	if err != nil {
		return nil, err
	}

	return stbw, nil
}

func (stbw *SearchTermBatchWriterV1) updateHighDocVersion(owner SearchIdxOwner, term string,
	termKey, indexKey *sscrypto.StrongSaltKey, updateID string) error {
	ssdi, err := OpenSearchSortDocIdxV1(owner, term, termKey, indexKey, updateID)
	if err == nil {
		err = nil
		for err == nil {
			var blk *SearchSortDocIdxBlkV1 = nil
			blk, err = ssdi.ReadNextBlock()
			if err != nil && err != io.EOF {
				return err
			}
			if blk != nil {
				for docID, docVer := range blk.docIDVerMap {
					if highVer, exist := stbw.highDocVerMap[docID]; exist && highVer < docVer {
						stbw.highDocVerMap[docID] = docVer
					}
				}
			}
		}
		ssdi.Close()
	} else if stbw.oldSti != nil {
		// TODO: handle the versioning properly
		sti := stbw.oldSti.(*SearchTermIdxV1)
		err = nil
		for err == nil {
			var blk *SearchTermIdxBlkV1 = nil
			blk, err = sti.ReadNextBlock()
			if err != nil && err != io.EOF {
				return err
			}
			if blk != nil {
				for docID, docVerOff := range blk.DocVerOffset {
					if highVer, exist := stbw.highDocVerMap[docID]; exist && highVer < docVerOff.Version {
						stbw.highDocVerMap[docID] = docVerOff.Version
					}
				}
			}
		}
		err = sti.Reset()
		if err != nil {
			return err
		}
	}

	return nil
}

func (stbw *SearchTermBatchWriterV1) ProcessSourceBlocks(sourceBlocks []*SearchTermIdxSourceBlockV1) ([]*SearchTermIdxBlkV1, error) {
	var err error
	returnBlocks := make([]*SearchTermIdxBlkV1, 0, len(sourceBlocks)*2)

	cloneSourceBlocks := make([]*SearchTermIdxSourceBlockV1, 0, len(sourceBlocks))

	// Remove all sources with lower version than highest document version
	for _, blk := range sourceBlocks {
		highVer, exist := stbw.highDocVerMap[blk.DocID]
		if exist && highVer <= blk.DocVer && !stbw.delDocMap[blk.DocID] {
			cloneSourceBlocks = append(cloneSourceBlocks, blk)
		}
	}

	// Read the old STI block
	if stbw.oldStiBlk == nil {
		err = stbw.getOldSearchTermIndexBlock()
		if err != nil && err != io.EOF {
			return nil, err
		}
	}

	// Create the new STI block to be written to
	if stbw.newStiBlk == nil {
		stbw.newStiBlk = CreateSearchTermIdxBlkV1(stbw.newSti.GetMaxBlockDataSize())
	}

	srcOffsetMap := make(map[*SearchTermIdxSourceBlockV1][]uint64)
	for _, srcBlk := range cloneSourceBlocks {
		if _, ok := srcBlk.TermOffset[stbw.Term]; ok {
			// Clone the offset array
			srcOffsetMap[srcBlk] = append([]uint64{}, srcBlk.TermOffset[stbw.Term]...)
		}
	}

	// Process incoming source blocks until all are empty
	for sourceNotEmpty := true; sourceNotEmpty; {
		var blks []*SearchTermIdxBlkV1 = nil

		sourceNotEmpty, blks, err = stbw.processSourceBlocks(cloneSourceBlocks, srcOffsetMap, copyLen)
		if err != nil {
			return nil, err
		}

		returnBlocks = append(returnBlocks, blks...)

		blks, err = stbw.processOldStiBlock(copyLen, false)
		if err != nil && err != io.EOF {
			return nil, err
		}

		returnBlocks = append(returnBlocks, blks...)
	} // for sourceNotEmpty := true; sourceNotEmpty;

	return returnBlocks, nil
}

func (stbw *SearchTermBatchWriterV1) processSourceBlocks(
	sourceBlocks []*SearchTermIdxSourceBlockV1,
	srcOffsetMap map[*SearchTermIdxSourceBlockV1][]uint64,
	copyLen int) (bool, []*SearchTermIdxBlkV1, error) {

	sourceNotEmpty := false
	var err error = nil
	returnBlocks := make([]*SearchTermIdxBlkV1, 0, 10)

	// Process the source blocks
	for _, srcBlk := range sourceBlocks {
		srcOffsets := srcOffsetMap[srcBlk]
		if srcOffsets == nil || len(srcOffsets) == 0 {
			continue
		}

		sourceNotEmpty = true
		count := utils.Min(len(srcOffsets), copyLen)
		offsets := srcOffsets[:count]
		err = stbw.newStiBlk.AddDocOffsets(srcBlk.DocID, srcBlk.DocVer, offsets)

		// Current block is full. Send it
		if err != nil {
			err = stbw.newSti.WriteNextBlock(stbw.newStiBlk)
			if err != nil {
				return sourceNotEmpty, returnBlocks, err
			}

			returnBlocks = append(returnBlocks, stbw.newStiBlk)

			stbw.newStiBlk = CreateSearchTermIdxBlkV1(stbw.newSti.GetMaxBlockDataSize())
			err = stbw.newStiBlk.AddDocOffsets(srcBlk.DocID, srcBlk.DocVer, offsets)
			if err != nil {
				return sourceNotEmpty, returnBlocks, err
			}
		}

		srcOffsetMap[srcBlk] = srcOffsets[count:]
	} // for _, srcBlk := range sourceBlocks

	return sourceNotEmpty, returnBlocks, nil
}

func (stbw *SearchTermBatchWriterV1) processOldStiBlock(copyLen int, writeFull bool) ([]*SearchTermIdxBlkV1, error) {
	var err error = nil
	returnBlocks := make([]*SearchTermIdxBlkV1, 0, 10)

	if stbw.oldStiBlk == nil {
		// No more old STI blocks to process
		return returnBlocks, io.EOF
	}

	// Process the old STI block
	for i := 0; i < len(stbw.oldStiBlkDocIDs); i++ {
		docID := stbw.oldStiBlkDocIDs[i]
		verOffset := stbw.oldStiBlk.DocVerOffset[docID]
		if verOffset == nil {
			stbw.oldStiBlkDocIDs = removeString(stbw.oldStiBlkDocIDs, i)
			i--
			continue
		}

		count := utils.Min(len(verOffset.Offsets), copyLen)
		offsets := verOffset.Offsets[:count]
		err = stbw.newStiBlk.AddDocOffsets(docID, verOffset.Version, offsets)
		if err != nil {
			if writeFull {
				err = stbw.newSti.WriteNextBlock(stbw.newStiBlk)
				if err != nil {
					return returnBlocks, err
				}

				returnBlocks = append(returnBlocks, stbw.newStiBlk)

				stbw.newStiBlk = CreateSearchTermIdxBlkV1(stbw.newSti.GetMaxBlockDataSize())
				err = stbw.newStiBlk.AddDocOffsets(docID, verOffset.Version, offsets)
				if err != nil {
					return returnBlocks, err
				}
			} else {
				// The new block is full. Don't bother processing anymore old STI block
				break
			}
		}

		// Remove the successfully processed offsets from the old STI block
		verOffset.Offsets = verOffset.Offsets[count:]
		if len(verOffset.Offsets) == 0 {
			stbw.oldStiBlkDocIDs = removeString(stbw.oldStiBlkDocIDs, i)
			i--
		}
	} // for i := 0; i < len(stbw.oldStiDocIDs); i++

	if len(stbw.oldStiBlkDocIDs) == 0 {
		err = stbw.getOldSearchTermIndexBlock()
		if err != nil && err != io.EOF {
			return returnBlocks, err
		}
	}

	return returnBlocks, err
}

func (stbw *SearchTermBatchWriterV1) getOldSearchTermIndexBlock() error {
	var err error = nil

	if stbw.oldSti != nil {
		// 1. Remove all document versions lower than high version
		// 2. Remove all documents in source block delete list
		// 3. Remove all documents in global delete list
		// 4. Set stbw.oldStiBlock
		// 5. Set stbw.oldStiBlkDocIDs

		stbw.oldStiBlk = nil
		stbw.oldStiBlkDocIDs = make([]string, 0)

		// TODO: handle the versioning properly
		oldSti := stbw.oldSti.(*SearchTermIdxV1)
		stbw.oldStiBlk, err = oldSti.ReadNextBlock()
		if err != nil && err != io.EOF {
			return err
		}
		if stbw.oldStiBlk != nil {
			blkDocIDs := make([]string, 0, len(stbw.oldStiBlk.DocVerOffset))
			for docID, docVerOff := range stbw.oldStiBlk.DocVerOffset {
				highVer, exist := stbw.highDocVerMap[docID]
				if (exist && highVer > docVerOff.Version) || stbw.delDocMap[docID] {
					delete(stbw.oldStiBlk.DocVerOffset, docID)
				} else {
					blkDocIDs = append(blkDocIDs, docID)
				}
			}
			stbw.oldStiBlkDocIDs = blkDocIDs
		}
	}

	return err
}

func (stbw *SearchTermBatchWriterV1) Close() ([]*SearchTermIdxBlkV1, error) {
	returnBlocks := make([]*SearchTermIdxBlkV1, 0, 10)
	var blks []*SearchTermIdxBlkV1 = nil
	var err, finalErr error = nil, nil

	if stbw.oldStiBlk != nil {
		for err != nil {
			blks, err = stbw.processOldStiBlock(copyLen, true)
			if err != nil && err != io.EOF {
				finalErr = err
			}
			if blks != nil && len(blks) > 0 {
				returnBlocks = append(returnBlocks, blks...)
			}
		}
		stbw.oldStiBlk = nil
	}

	if stbw.newStiBlk != nil && !stbw.newStiBlk.IsEmpty() {
		err = stbw.newSti.WriteNextBlock(stbw.newStiBlk)
		if err != nil {
			finalErr = err
		} else {
			returnBlocks = append(returnBlocks, stbw.newStiBlk)
		}
		stbw.newStiBlk = nil
	}

	if stbw.oldSti != nil {
		err = stbw.oldSti.Close()
		if err != nil {
			finalErr = err
		}
	}

	if stbw.newSti != nil {
		err = stbw.newSti.Close()
		if err != nil {
			finalErr = err
		}
	}

	return returnBlocks, finalErr
}

func removeSourceBlock(blocks []*SearchTermIdxSourceBlockV1, i int) []*SearchTermIdxSourceBlockV1 {
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
