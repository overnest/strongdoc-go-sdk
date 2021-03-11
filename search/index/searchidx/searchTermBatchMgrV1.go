package searchidx

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/emirpasic/gods/trees/binaryheap"
	"github.com/overnest/strongdoc-go-sdk/utils"

	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

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

type SearchTermBatchSources struct {
	Term       string
	TermKey    *sscrypto.StrongSaltKey
	IndexKey   *sscrypto.StrongSaltKey
	AddSources []SearchTermIdxSourceV1
	DelSources []SearchTermIdxSourceV1
	delDocs    *DeletedDocsV1
}

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
	wa := a.(*SearchTermBatchSources)
	wb := b.(*SearchTermBatchSources)
	return strings.Compare(wa.Term, wb.Term)
}

func CreateSearchTermBatchMgrV1(owner SearchIdxOwner, sources []SearchTermIdxSourceV1,
	termKey, indexKey *sscrypto.StrongSaltKey, delDocs *DeletedDocsV1) (*SearchTermBatchMgrV1, error) {

	mgr := &SearchTermBatchMgrV1{
		Owner:    owner,
		TermKey:  termKey,
		IndexKey: indexKey,
		delDocs:  delDocs,
		termHeap: binaryheap.NewWith(batchSourceComparator)}

	termSourcesMap := make(map[string]*SearchTermBatchSources) // term -> termSources
	for _, source := range sources {
		for _, term := range source.GetAddTerms() {
			stbs := termSourcesMap[term]
			if stbs == nil {
				stbs = &SearchTermBatchSources{
					Term:       term,
					TermKey:    termKey,
					IndexKey:   indexKey,
					AddSources: make([]SearchTermIdxSourceV1, 0, 10),
					DelSources: make([]SearchTermIdxSourceV1, 0, 10),
					delDocs:    delDocs}
				termSourcesMap[term] = stbs
				mgr.termHeap.Push(stbs)
			}
			stbs.AddSources = append(stbs.AddSources, source)
		}

		for _, term := range source.GetDelTerms() {
			stbs := termSourcesMap[term]
			if stbs == nil {
				stbs = &SearchTermBatchSources{
					Term:       term,
					TermKey:    termKey,
					IndexKey:   indexKey,
					AddSources: make([]SearchTermIdxSourceV1, 0, 10),
					DelSources: make([]SearchTermIdxSourceV1, 0, 10),
					delDocs:    delDocs}
				termSourcesMap[term] = stbs
				mgr.termHeap.Push(stbs)
			}
			stbs.DelSources = append(stbs.DelSources, source)
		}
	}

	return mgr, nil
}

func (mgr *SearchTermBatchMgrV1) GetNextTermBatch(batchSize int) (*SearchTermBatchV1, error) {
	sources := make([]*SearchTermBatchSources, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		s, _ := mgr.termHeap.Pop()
		if s == nil {
			break
		}

		stbs := s.(*SearchTermBatchSources)
		sources = append(sources, stbs)
	}

	return CreateSearchTermBatchV1(mgr.Owner, sources)
}

func (mgr *SearchTermBatchMgrV1) ProcessNextTermBatch(batchSize int) (map[string]error, error) {
	batchResult := make(map[string]error)

	termBatch, err := mgr.GetNextTermBatch(batchSize)
	if err != nil {
		return batchResult, err
	}
	defer termBatch.Close()

	if termBatch.IsEmpty() {
		return batchResult, io.EOF
	}

	return termBatch.ProcessTermBatch()
}

func (mgr *SearchTermBatchMgrV1) ProcessAllTermBatches(batchSize int) (map[string]error, error) {
	finalResult := make(map[string]error)

	var err error = nil
	for err == nil {
		var batchResult map[string]error

		batchResult, err = mgr.ProcessNextTermBatch(batchSize)
		if err != nil && err != io.EOF {
			return finalResult, err
		}

		for term, err := range batchResult {
			finalResult[term] = err
		}
	}

	return finalResult, err
}

func (mgr *SearchTermBatchMgrV1) Close() error {
	return nil
}

//////////////////////////////////////////////////////////////////
//
//                    Search Term Batch V1
//
//////////////////////////////////////////////////////////////////

type SearchTermBatchV1 struct {
	Owner           SearchIdxOwner
	SourceList      []SearchTermIdxSourceV1
	TermToWriter    map[string]*SearchTermIdxWriterV1 // term -> SearchTermIdxWriterV1
	sourceToWriters map[SearchTermIdxSourceV1][]*SearchTermIdxWriterV1
	termList        []string
}

func CreateSearchTermBatchV1(owner SearchIdxOwner, sources []*SearchTermBatchSources) (*SearchTermBatchV1, error) {
	batch := &SearchTermBatchV1{
		Owner:           owner,
		SourceList:      nil,
		TermToWriter:    make(map[string]*SearchTermIdxWriterV1),
		sourceToWriters: make(map[SearchTermIdxSourceV1][]*SearchTermIdxWriterV1),
		termList:        nil,
	}

	for _, source := range sources {

		stiw, err := CreateSearchTermIdxWriterV1(
			owner, source.Term, source.AddSources, source.DelSources,
			source.TermKey, source.IndexKey, source.delDocs)
		if err != nil {
			return nil, err
		}
		batch.TermToWriter[source.Term] = stiw
	}

	sourceMap := make(map[SearchTermIdxSourceV1]map[*SearchTermIdxWriterV1]bool)
	for _, stiw := range batch.TermToWriter {
		for _, source := range stiw.AddSources {
			stiwMap := sourceMap[source]
			if stiwMap == nil {
				stiwMap = make(map[*SearchTermIdxWriterV1]bool)
				sourceMap[source] = stiwMap
			}
			stiwMap[stiw] = true
		}
	}

	for stis, stiwMap := range sourceMap {
		stiwList := make([]*SearchTermIdxWriterV1, 0, len(stiwMap))
		for stbi := range stiwMap {
			stiwList = append(stiwList, stbi)
		}
		batch.sourceToWriters[stis] = stiwList
	}

	batch.SourceList = make([]SearchTermIdxSourceV1, 0, len(batch.sourceToWriters))
	for sis := range batch.sourceToWriters {
		batch.SourceList = append(batch.SourceList, sis)
	}

	batch.termList = make([]string, 0, len(batch.TermToWriter))
	for term := range batch.TermToWriter {
		batch.termList = append(batch.termList, term)
	}
	sort.Strings(batch.termList)

	return batch, nil
}

func (batch *SearchTermBatchV1) ProcessTermBatch() (map[string]error, error) {
	respMap := make(map[string]error)

	t1 := time.Now()
	stiResp, err := batch.processStiAll()
	if err != nil {
		return respMap, err
	}

	t2 := time.Now()
	stiSuccess := make([]*SearchTermIdxWriterV1, 0, len(stiResp))
	for stiw, err := range stiResp {
		respMap[stiw.Term] = err
		if err == nil {
			stiSuccess = append(stiSuccess, stiw)
		}
	}

	t3 := time.Now()
	ssdiResp, err := batch.processSsdiAll(stiSuccess)
	for stiw, err := range ssdiResp {
		respMap[stiw.Term] = err
	}

	t4 := time.Now()

	// PSL DEBUG
	if true {
		fmt.Println("Process STI", t2.Sub(t1).Milliseconds(), "ms")
		fmt.Println("Process SSDI", t4.Sub(t3).Milliseconds(), "ms")
		fmt.Println("--------------------------")
	}

	return respMap, nil
}

func (batch *SearchTermBatchV1) processStiBatch() (map[*SearchTermIdxWriterV1]*SearchTermIdxWriterRespV1, error) {

	stiwToBlks := make(map[*SearchTermIdxWriterV1][]*SearchTermIdxSourceBlockV1)

	for _, source := range batch.SourceList {
		blk, err := source.GetNextSourceBlock(batch.termList)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if blk != nil && len(blk.TermOffset) > 0 {
			for _, stiw := range batch.sourceToWriters[source] {
				blkList := stiwToBlks[stiw]
				if blkList == nil {
					blkList = make([]*SearchTermIdxSourceBlockV1, 0, len(stiw.AddSources))
				}
				stiwToBlks[stiw] = append(blkList, blk)
			}
		}
	}

	stiwToChan := make(map[*SearchTermIdxWriterV1](chan *SearchTermIdxWriterRespV1))
	for stiw, blkList := range stiwToBlks {
		if blkList != nil && len(blkList) > 0 {
			stiwChan := make(chan *SearchTermIdxWriterRespV1)
			stiwToChan[stiw] = stiwChan

			// This executes in a separate thread
			go func(stiw *SearchTermIdxWriterV1, blkList []*SearchTermIdxSourceBlockV1,
				stiwChan chan<- *SearchTermIdxWriterRespV1) {
				defer close(stiwChan)

				stibs, err := stiw.ProcessSourceBlocks(blkList)
				stiwChan <- &SearchTermIdxWriterRespV1{stibs, err}
			}(stiw, blkList, stiwChan)
		}
	}

	respMap := make(map[*SearchTermIdxWriterV1]*SearchTermIdxWriterRespV1)
	for stiw, stiwChan := range stiwToChan {
		respMap[stiw] = <-stiwChan
	}

	// Nothing left to process
	if len(respMap) == 0 {
		return respMap, io.EOF
	}

	return respMap, nil
}

func (batch *SearchTermBatchV1) processStiAll() (map[*SearchTermIdxWriterV1]error, error) {
	respMap := make(map[*SearchTermIdxWriterV1]error)
	for _, source := range batch.SourceList {
		source.Reset()
	}

	for true {
		stiBlocksResp, err := batch.processStiBatch()
		if err != nil && err != io.EOF {
			return nil, err
		}

		if stiBlocksResp != nil && len(stiBlocksResp) > 0 {
			for stiw, resp := range stiBlocksResp {
				if resp != nil {
					// If there is already an error for this STIW, do not overwrite
					if respMap[stiw] == nil {
						respMap[stiw] = resp.Error
					}
				} else {
					respMap[stiw] = nil
				}
			}
		}

		// Should be io.EOF
		if err != nil {
			break
		}
	}

	for _, stiw := range batch.TermToWriter {
		// TODO: Do this in parallel
		stiw.Close()
	}

	return respMap, nil
}

func (batch *SearchTermBatchV1) processSsdiAll(stiwList []*SearchTermIdxWriterV1) (map[*SearchTermIdxWriterV1]error, error) {
	ssdiToChan := make(map[*SearchTermIdxWriterV1](chan error))

	for _, stiw := range stiwList {
		ssdiChan := make(chan error)
		ssdiToChan[stiw] = ssdiChan

		// This executes in a separate thread
		go func(stiw *SearchTermIdxWriterV1, ssdiChan chan<- error) {
			defer close(ssdiChan)

			ssdi, err := CreateSearchSortDocIdxV1(batch.Owner, stiw.Term,
				stiw.GetUpdateID(), stiw.TermKey, stiw.IndexKey)
			if err != nil {
				ssdiChan <- err
				return
			}
			defer ssdi.Close()

			for err == nil {
				var blk *SearchSortDocIdxBlkV1
				blk, err = ssdi.WriteNextBlock()

				_ = blk
				// fmt.Println(stiw.Term, blk.DocIDVers)

				if err != nil && err != io.EOF {
					ssdiChan <- err
					return
				}
			}

			ssdiChan <- nil
		}(stiw, ssdiChan)
	}

	respMap := make(map[*SearchTermIdxWriterV1]error)
	for stiw, ssdiChan := range ssdiToChan {
		respMap[stiw] = <-ssdiChan
	}

	return respMap, nil
}

func (batch *SearchTermBatchV1) IsEmpty() bool {
	return (len(batch.TermToWriter) == 0)
}

func (batch *SearchTermBatchV1) Close() error {
	// No Op
	return nil
}

//////////////////////////////////////////////////////////////////
//
//                 Search Term Batch Writer V1
//
//////////////////////////////////////////////////////////////////

type SearchTermIdxWriterRespV1 struct {
	Blocks []*SearchTermIdxBlkV1
	Error  error
}

type SearchTermIdxWriterV1 struct {
	Term            string
	Owner           SearchIdxOwner
	AddSources      []SearchTermIdxSourceV1
	DelSources      []SearchTermIdxSourceV1
	TermKey         *sscrypto.StrongSaltKey
	IndexKey        *sscrypto.StrongSaltKey
	updateID        string
	delDocMap       map[string]bool   // DocID -> boolean
	highDocVerMap   map[string]uint64 // DocID -> DocVer
	newSti          *SearchTermIdxV1
	oldSti          SearchTermIdx
	newStiBlk       *SearchTermIdxBlkV1
	oldStiBlk       *SearchTermIdxBlkV1
	oldStiBlkDocIDs []string
}

func CreateSearchTermIdxWriterV1(owner SearchIdxOwner, term string,
	addSource, delSource []SearchTermIdxSourceV1,
	termKey, indexKey *sscrypto.StrongSaltKey, delDocs *DeletedDocsV1) (*SearchTermIdxWriterV1, error) {

	var err error

	if addSource == nil {
		addSource = make([]SearchTermIdxSourceV1, 0, 10)
	}
	if delSource == nil {
		delSource = make([]SearchTermIdxSourceV1, 0, 10)
	}

	stiw := &SearchTermIdxWriterV1{
		Owner:           owner,
		Term:            term,
		AddSources:      addSource,
		DelSources:      delSource,
		TermKey:         termKey,
		IndexKey:        indexKey,
		updateID:        "",
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
		stiw.delDocMap[ds.GetDocID()] = true
	}

	if delDocs != nil {
		for _, docID := range delDocs.DelDocs {
			stiw.delDocMap[docID] = true
		}
	}

	// Find the highest version of each DocID. This is so we can remove
	// any DocID with lower version
	for _, stis := range addSource {
		if ver, exist := stiw.highDocVerMap[stis.GetDocID()]; !exist || ver < stis.GetDocVer() {
			stiw.highDocVerMap[stis.GetDocID()] = stis.GetDocVer()
		}
	}
	for _, stis := range delSource {
		if ver, exist := stiw.highDocVerMap[stis.GetDocID()]; !exist || ver < stis.GetDocVer() {
			stiw.highDocVerMap[stis.GetDocID()] = stis.GetDocVer()
		}
	}

	// Open previous STI if there is one
	updateID, err := GetLatestUpdateIDV1(owner, term, termKey)
	if updateID != "" {
		stiw.oldSti, err = OpenSearchTermIdxV1(owner, term, termKey, indexKey, updateID)
		if err != nil {
			stiw.oldSti = nil
		}
		err = stiw.updateHighDocVersion(owner, term, termKey, indexKey, updateID)
		if err != nil {
			return nil, err
		}
	}

	stiw.newSti, err = CreateSearchTermIdxV1(owner, term, termKey, indexKey, nil, nil)
	if err != nil {
		return nil, err
	}

	stiw.updateID = stiw.newSti.updateID
	return stiw, nil
}

func (stiw *SearchTermIdxWriterV1) updateHighDocVersion(owner SearchIdxOwner, term string,
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
					if highVer, exist := stiw.highDocVerMap[docID]; exist && highVer < docVer {
						stiw.highDocVerMap[docID] = docVer
					}
				}
			}
		}
		ssdi.Close()
	} else if stiw.oldSti != nil {
		// TODO: handle the versioning properly
		sti := stiw.oldSti.(*SearchTermIdxV1)
		err = nil
		for err == nil {
			var blk *SearchTermIdxBlkV1 = nil
			blk, err = sti.ReadNextBlock()
			if err != nil && err != io.EOF {
				return err
			}
			if blk != nil {
				for docID, docVerOff := range blk.DocVerOffset {
					if highVer, exist := stiw.highDocVerMap[docID]; exist && highVer < docVerOff.Version {
						stiw.highDocVerMap[docID] = docVerOff.Version
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

func (stiw *SearchTermIdxWriterV1) ProcessSourceBlocks(sourceBlocks []*SearchTermIdxSourceBlockV1) ([]*SearchTermIdxBlkV1, error) {

	var err error
	returnBlocks := make([]*SearchTermIdxBlkV1, 0, len(sourceBlocks)*2)
	cloneSourceBlocks := make([]*SearchTermIdxSourceBlockV1, 0, len(sourceBlocks))

	// Remove all sources with lower version than highest document version
	for _, blk := range sourceBlocks {
		highVer, exist := stiw.highDocVerMap[blk.DocID]
		if exist && highVer <= blk.DocVer && !stiw.delDocMap[blk.DocID] {
			cloneSourceBlocks = append(cloneSourceBlocks, blk)
		}
	}

	// Read the old STI block
	if stiw.oldStiBlk == nil {
		err = stiw.getOldSearchTermIndexBlock()
		if err != nil && err != io.EOF {
			return nil, err
		}
	}

	// Create the new STI block to be written to
	if stiw.newStiBlk == nil {
		stiw.newStiBlk = CreateSearchTermIdxBlkV1(stiw.newSti.GetMaxBlockDataSize())
	}

	srcOffsetMap := make(map[*SearchTermIdxSourceBlockV1][]uint64)
	for _, srcBlk := range cloneSourceBlocks {
		if _, ok := srcBlk.TermOffset[stiw.Term]; ok {
			// Clone the offset array
			srcOffsetMap[srcBlk] = append([]uint64{}, srcBlk.TermOffset[stiw.Term]...)
		}
	}

	// Process incoming source blocks until all are empty
	for sourceNotEmpty := true; sourceNotEmpty; {
		var blks []*SearchTermIdxBlkV1 = nil

		sourceNotEmpty, blks, err = stiw.processSourceBlocks(cloneSourceBlocks, srcOffsetMap, copyLen)
		if err != nil {
			return nil, err
		}

		returnBlocks = append(returnBlocks, blks...)

		blks, err = stiw.processOldStiBlock(copyLen, false)
		if err != nil && err != io.EOF {
			return nil, err
		}

		returnBlocks = append(returnBlocks, blks...)
	} // for sourceNotEmpty := true; sourceNotEmpty;

	return returnBlocks, nil
}

func (stiw *SearchTermIdxWriterV1) processSourceBlocks(
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
		err = stiw.newStiBlk.AddDocOffsets(srcBlk.DocID, srcBlk.DocVer, offsets)

		// Current block is full. Send it
		if err != nil {
			err = stiw.newSti.WriteNextBlock(stiw.newStiBlk)
			if err != nil {
				return sourceNotEmpty, returnBlocks, err
			}

			returnBlocks = append(returnBlocks, stiw.newStiBlk)

			stiw.newStiBlk = CreateSearchTermIdxBlkV1(stiw.newSti.GetMaxBlockDataSize())
			err = stiw.newStiBlk.AddDocOffsets(srcBlk.DocID, srcBlk.DocVer, offsets)
			if err != nil {
				return sourceNotEmpty, returnBlocks, err
			}
		}

		srcOffsetMap[srcBlk] = srcOffsets[count:]
	} // for _, srcBlk := range sourceBlocks

	return sourceNotEmpty, returnBlocks, nil
}

func (stiw *SearchTermIdxWriterV1) processOldStiBlock(copyLen int, writeFull bool) ([]*SearchTermIdxBlkV1, error) {
	var err error = nil
	returnBlocks := make([]*SearchTermIdxBlkV1, 0, 10)

	if stiw.oldStiBlk == nil {
		// No more old STI blocks to process
		return returnBlocks, io.EOF
	}

	// Process the old STI block
	for i := 0; i < len(stiw.oldStiBlkDocIDs); i++ {
		docID := stiw.oldStiBlkDocIDs[i]
		verOffset := stiw.oldStiBlk.DocVerOffset[docID]
		if verOffset == nil {
			stiw.oldStiBlkDocIDs = removeString(stiw.oldStiBlkDocIDs, i)
			i--
			continue
		}

		count := utils.Min(len(verOffset.Offsets), copyLen)
		offsets := verOffset.Offsets[:count]
		err = stiw.newStiBlk.AddDocOffsets(docID, verOffset.Version, offsets)
		if err != nil {
			if writeFull {
				err = stiw.newSti.WriteNextBlock(stiw.newStiBlk)
				if err != nil {
					return returnBlocks, err
				}

				returnBlocks = append(returnBlocks, stiw.newStiBlk)

				stiw.newStiBlk = CreateSearchTermIdxBlkV1(stiw.newSti.GetMaxBlockDataSize())
				err = stiw.newStiBlk.AddDocOffsets(docID, verOffset.Version, offsets)
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
			stiw.oldStiBlkDocIDs = removeString(stiw.oldStiBlkDocIDs, i)
			i--
		}
	} // for i := 0; i < len(stiw.oldStiDocIDs); i++

	if len(stiw.oldStiBlkDocIDs) == 0 {
		err = stiw.getOldSearchTermIndexBlock()
		if err != nil && err != io.EOF {
			return returnBlocks, err
		}
	}

	return returnBlocks, err
}

func (stiw *SearchTermIdxWriterV1) getOldSearchTermIndexBlock() error {
	var err error = nil

	if stiw.oldSti != nil {
		// 1. Remove all document versions lower than high version
		// 2. Remove all documents in source block delete list
		// 3. Remove all documents in global delete list
		// 4. Set stiw.oldStiBlock
		// 5. Set stiw.oldStiBlkDocIDs

		stiw.oldStiBlk = nil
		stiw.oldStiBlkDocIDs = make([]string, 0)

		// TODO: handle the versioning properly
		oldSti := stiw.oldSti.(*SearchTermIdxV1)
		stiw.oldStiBlk, err = oldSti.ReadNextBlock()
		if err != nil && err != io.EOF {
			return err
		}
		if stiw.oldStiBlk != nil {
			blkDocIDs := make([]string, 0, len(stiw.oldStiBlk.DocVerOffset))
			for docID, docVerOff := range stiw.oldStiBlk.DocVerOffset {
				highVer, exist := stiw.highDocVerMap[docID]
				if (exist && highVer > docVerOff.Version) || stiw.delDocMap[docID] {
					delete(stiw.oldStiBlk.DocVerOffset, docID)
				} else {
					blkDocIDs = append(blkDocIDs, docID)
				}
			}
			stiw.oldStiBlkDocIDs = blkDocIDs
		}
	}

	return err
}

func (stiw *SearchTermIdxWriterV1) GetUpdateID() string {
	return stiw.updateID
}

func (stiw *SearchTermIdxWriterV1) Close() ([]*SearchTermIdxBlkV1, error) {
	returnBlocks := make([]*SearchTermIdxBlkV1, 0, 10)
	var blks []*SearchTermIdxBlkV1 = nil
	var err, finalErr error = nil, nil

	if stiw.oldStiBlk != nil {
		for err != nil {
			blks, err = stiw.processOldStiBlock(copyLen, true)
			if err != nil && err != io.EOF {
				finalErr = err
			}
			if blks != nil && len(blks) > 0 {
				returnBlocks = append(returnBlocks, blks...)
			}
		}
		stiw.oldStiBlk = nil
	}

	if stiw.newStiBlk != nil && !stiw.newStiBlk.IsEmpty() {
		err = stiw.newSti.WriteNextBlock(stiw.newStiBlk)
		if err != nil {
			finalErr = err
		} else {
			returnBlocks = append(returnBlocks, stiw.newStiBlk)
		}
		stiw.newStiBlk = nil
	}

	if stiw.oldSti != nil {
		err = stiw.oldSti.Close()
		if err != nil {
			finalErr = err
		}
	}

	if stiw.newSti != nil {
		err = stiw.newSti.Close()
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
