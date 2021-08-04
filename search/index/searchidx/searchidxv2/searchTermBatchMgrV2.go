package searchidxv2

import (
	"github.com/emirpasic/gods/trees/binaryheap"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"io"
	"sort"
	"strings"
)

const copyLen = 10 //TODO

func removeString(stringList []string, i int) []string {
	if stringList != nil && i >= 0 && i < len(stringList) {
		stringList[i] = stringList[len(stringList)-1]
		stringList = stringList[:len(stringList)-1]
	}
	return stringList
}

//////////////////////////////////////////////////////////////////
//
//                   Search HashedTerm Batch Manager V2
//
//////////////////////////////////////////////////////////////////

type SearchTermBatchMgrV2 struct {
	Owner            common.SearchIdxOwner
	SearchIdxSources []SearchTermIdxSourceV2
	TermKey          *sscrypto.StrongSaltKey
	IndexKey         *sscrypto.StrongSaltKey
	delDocs          *DeletedDocsV2 // global delete list
	termHeap         *binaryheap.Heap
}

type SearchTermBatchSources struct {
	Term       string // plain term
	AddSources []SearchTermIdxSourceV2
	DelSources []SearchTermIdxSourceV2
	delDocs    *DeletedDocsV2 // global delete list
}

type SearchTermBatchElement struct {
	HashedTerm  string
	TermKey     *sscrypto.StrongSaltKey
	IndexKey    *sscrypto.StrongSaltKey
	TermSources []*SearchTermBatchSources
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
	wa := a.(*SearchTermBatchElement)
	wb := b.(*SearchTermBatchElement)
	return strings.Compare(wa.HashedTerm, wb.HashedTerm) // not necessary to sort
}

func CreateSearchTermBatchMgrV2(owner common.SearchIdxOwner, sources []SearchTermIdxSourceV2,
	termKey, indexKey *sscrypto.StrongSaltKey, delDocs *DeletedDocsV2) (*SearchTermBatchMgrV2, error) {

	mgr := &SearchTermBatchMgrV2{
		Owner:    owner,
		TermKey:  termKey,
		IndexKey: indexKey,
		delDocs:  delDocs,
		termHeap: binaryheap.NewWith(batchSourceComparator)}

	termSourcesMap := make(map[string]*SearchTermBatchSources) // term -> termSources
	for _, source := range sources {
		for _, term := range source.GetAllTerms() {
			stbs := termSourcesMap[term]
			if stbs == nil {
				stbs = &SearchTermBatchSources{
					Term:       term,
					AddSources: make([]SearchTermIdxSourceV2, 0, 10),
					DelSources: make([]SearchTermIdxSourceV2, 0, 10),
					delDocs:    delDocs}
				termSourcesMap[term] = stbs
			}
			stbs.AddSources = append(stbs.AddSources, source)
		}

		for _, term := range source.GetDelTerms() {
			stbs := termSourcesMap[term]
			if stbs == nil {
				stbs = &SearchTermBatchSources{
					Term:       term,
					AddSources: make([]SearchTermIdxSourceV2, 0, 10),
					DelSources: make([]SearchTermIdxSourceV2, 0, 10),
					delDocs:    delDocs}
				termSourcesMap[term] = stbs
			}
			stbs.DelSources = append(stbs.DelSources, source)
		}
	}

	batchElementMap := make(map[string]*SearchTermBatchElement) // hashedTerm -> batchElement
	for term, termSources := range termSourcesMap {
		hashedTerm := common.HashTerm(term)
		batchElement := batchElementMap[hashedTerm]
		if batchElement == nil {
			batchElement = &SearchTermBatchElement{
				HashedTerm: hashedTerm,
				IndexKey:   indexKey,
				TermKey:    termKey,
			}
			batchElementMap[hashedTerm] = batchElement
			mgr.termHeap.Push(batchElement)
		}
		batchElement.TermSources = append(batchElement.TermSources, termSources)
	}
	return mgr, nil
}

//////////////////////////////////////////////////////////////////
//
//                    Search HashedTerm Batch V2
//
//////////////////////////////////////////////////////////////////
type SearchTermIdxTermInfo struct {
	term       string // plain term
	AddSources []SearchTermIdxSourceV2
	DelSources []SearchTermIdxSourceV2
	delDocMap  map[string]bool //
}
type SearchTermBatchV2 struct {
	Owner                     common.SearchIdxOwner                              // owner
	SourceList                []SearchTermIdxSourceV2                            // all docs
	termList                  []string                                           // all terms
	hashedTermList            []string                                           // all hashed terms
	HashedTermToWriter        map[string]*SearchTermIdxWriterV2                  // hashedTerm -> index writer
	HashedTermToTerms         map[string][]*SearchTermIdxTermInfo                // hashedTerm -> terms mapped to this hashedTerm
	sourceToHashTerms         map[SearchTermIdxSourceV2][]string                 // source -> hashedTerms
	sourceToHashedTermWriters map[SearchTermIdxSourceV2][]*SearchTermIdxWriterV2 // source -> hashedTerm writers
}

func (mgr *SearchTermBatchMgrV2) GetNextTermBatch(sdc client.StrongDocClient, batchSize int) (*SearchTermBatchV2, error) {

	// pop out batch elements
	var sources []*SearchTermBatchElement
	for i := 0; i < batchSize; i++ {
		s, _ := mgr.termHeap.Pop()
		if s == nil {
			break
		}
		source := s.(*SearchTermBatchElement)
		sources = append(sources, source)
	}

	return CreateSearchTermBatchV2(sdc, mgr.Owner, sources)
}

// TODO: optimization
func CreateSearchTermBatchV2(sdc client.StrongDocClient, owner common.SearchIdxOwner,
	batchElements []*SearchTermBatchElement) (*SearchTermBatchV2, error) {
	batch := &SearchTermBatchV2{
		Owner:              owner,
		SourceList:         nil,                                     // all docs
		HashedTermToWriter: make(map[string]*SearchTermIdxWriterV2), // hashedTerm -> writer
		sourceToHashTerms:  nil,
		termList:           nil, // all terms
		hashedTermList:     nil, // all hashed terms
		HashedTermToTerms:  nil,
	}

	sourceToHashTermsMap := make(map[SearchTermIdxSourceV2]map[string]bool)
	for _, batchElement := range batchElements {
		hashedTerm := batchElement.HashedTerm
		for _, term := range batchElement.TermSources {
			for _, source := range term.AddSources {
				if _, ok := sourceToHashTermsMap[source]; !ok {
					sourceToHashTermsMap[source] = make(map[string]bool)
				}
				sourceToHashTermsMap[source][hashedTerm] = true
			}
		}
	}

	// source -> hashTerms
	sourceToHashTerms := make(map[SearchTermIdxSourceV2][]string)
	for source, hashTermsMap := range sourceToHashTermsMap {
		var hashTerms []string
		if hashTermsMap != nil && len(hashTermsMap) > 0 {
			for hashTerm, _ := range hashTermsMap {
				hashTerms = append(hashTerms, hashTerm)
			}
		}
		sourceToHashTerms[source] = hashTerms
	}
	batch.sourceToHashTerms = sourceToHashTerms

	// sourceList
	batch.SourceList = make([]SearchTermIdxSourceV2, 0, len(batch.sourceToHashTerms))
	for sis := range batch.sourceToHashTerms {
		batch.SourceList = append(batch.SourceList, sis)
	}

	// HashedTermToTerms
	hashedTermToTerms := make(map[string][]*SearchTermIdxTermInfo)
	hashTermToAllDelDocsMap := make(map[string]map[string]map[string]bool)
	for _, batchElement := range batchElements {
		var terms []*SearchTermIdxTermInfo
		hashTermToAllDelDocsMap[batchElement.HashedTerm] = make(map[string]map[string]bool)
		for _, termSource := range batchElement.TermSources {

			delDocMap := make(map[string]bool)
			if termSource.DelSources != nil {
				for _, ds := range termSource.DelSources {
					delDocMap[ds.GetDocID()] = true
				}
			}

			if termSource.delDocs != nil {
				for _, docID := range termSource.delDocs.DelDocs {
					delDocMap[docID] = true
				}
			}

			termInfo := &SearchTermIdxTermInfo{
				term:       termSource.Term,
				AddSources: termSource.AddSources,
				DelSources: termSource.DelSources,
				delDocMap:  delDocMap,
			}
			terms = append(terms, termInfo)
			hashTermToAllDelDocsMap[batchElement.HashedTerm][termSource.Term] = delDocMap
		}
		hashedTermToTerms[batchElement.HashedTerm] = terms

	}
	batch.HashedTermToTerms = hashedTermToTerms

	// HashedTermToWriter
	var allTerms []string
	var allHashedTerms []string
	for _, batchElement := range batchElements {

		allHashedTerms = append(allHashedTerms, batchElement.HashedTerm)
		// create term index writer for every hashedTerm
		writer, err := CreateSearchTermIdxWriterV2(sdc, owner,
			batchElement.HashedTerm, batchElement.TermSources,
			hashTermToAllDelDocsMap[batchElement.HashedTerm],
			batchElement.TermKey, batchElement.IndexKey)
		if err != nil {
			return nil, err
		}
		// collect all terms
		for _, term := range batchElement.TermSources {
			allTerms = append(allTerms, term.Term)

		}

		batch.HashedTermToWriter[batchElement.HashedTerm] = writer
	}
	//sort.Strings(allTerms)
	batch.termList = allTerms
	sort.Strings(allHashedTerms)
	batch.hashedTermList = allHashedTerms

	sourceToHashedTermWriters := make(map[SearchTermIdxSourceV2][]*SearchTermIdxWriterV2)
	for source, hashTermsMap := range sourceToHashTermsMap {
		var hashTermWriters []*SearchTermIdxWriterV2
		if hashTermsMap != nil && len(hashTermsMap) > 0 {
			for hashTerm, _ := range hashTermsMap {
				hashTermWriter := batch.HashedTermToWriter[hashTerm]
				hashTermWriters = append(hashTermWriters, hashTermWriter)
			}
		}
		sourceToHashedTermWriters[source] = hashTermWriters
	}
	batch.sourceToHashedTermWriters = sourceToHashedTermWriters

	return batch, nil
}

func (batch *SearchTermBatchV2) ProcessTermBatch(sdc client.StrongDocClient, event *utils.TimeEvent) (map[string]error, error) {
	respMap := make(map[string]error)

	e1 := utils.AddSubEvent(event, "processStiAll")
	processDocs, stiResp, err := batch.processStiAll(e1)
	if err != nil {
		return respMap, err
	}
	utils.EndEvent(e1)

	stiSuccess := make([]*SearchTermIdxWriterV2, 0, len(stiResp))
	for stiw, err := range stiResp {
		respMap[stiw.HashedTerm] = utils.FirstError(respMap[stiw.HashedTerm], err)
		if err == nil {
			stiSuccess = append(stiSuccess, stiw)
		}
	}

	e2 := utils.AddSubEvent(event, "processSsdiAll")
	ssdiResp, err := batch.processSsdiAll(sdc, stiSuccess, processDocs)
	if err != nil {
		return respMap, err
	}

	for stiw, err := range ssdiResp {
		respMap[stiw.HashedTerm] = utils.FirstError(respMap[stiw.HashedTerm], err)
	}
	utils.EndEvent(e2)

	return respMap, nil
}

// ------------------ Process SSDI------------------
func (batch *SearchTermBatchV2) processSsdiAll(sdc client.StrongDocClient,
	stiwList []*SearchTermIdxWriterV2,
	processedDocs map[string]map[string]map[string]uint64) (map[*SearchTermIdxWriterV2]error, error) {

	ssdiToChan := make(map[*SearchTermIdxWriterV2]chan error)

	for _, stiw := range stiwList {
		ssdiChan := make(chan error)
		ssdiToChan[stiw] = ssdiChan

		// process each hashedTerm in a separate thread
		go func(stiw *SearchTermIdxWriterV2,
			docs map[string]map[string]uint64, // [term] -> [[docID] -> ver]
			ssdiChan chan<- error) {
			defer close(ssdiChan)
			ssdi, err := CreateSearchSortDocIdxV2(sdc, batch.Owner, stiw.HashedTerm,
				stiw.GetUpdateID(), stiw.TermKey, stiw.IndexKey, docs)
			if err != nil {
				ssdiChan <- err
				return
			}
			defer ssdi.Close()

			for err == nil {
				var blk *SearchSortDocIdxBlkV2
				blk, err = ssdi.WriteNextBlock()

				_ = blk

				if err != nil && err != io.EOF {
					ssdiChan <- err
					return
				}
			}

			ssdi.Close()
			ssdiChan <- nil
		}(stiw, processedDocs[stiw.HashedTerm], ssdiChan)

	}

	respMap := make(map[*SearchTermIdxWriterV2]error)
	for stiw, ssdiChan := range ssdiToChan {
		respMap[stiw] = <-ssdiChan
	}

	return respMap, nil
}

// ------------------ Process STI------------------
type SearchTermBlockRespV2 struct {
	Block *SearchTermIdxSourceBlockV2
	Error error
}

type SearchTermReadingRespV2 struct {
	//hashedTermToBlks map[string][]*SearchTermIdxSourceBlockV2
	hashedTermWriterToBlks map[*SearchTermIdxWriterV2][]*SearchTermIdxSourceBlockV2
	Error                  error
}

type SearchTermWritingRespV2 struct {
	hashedTermWriterToResp map[*SearchTermIdxWriterV2]*SearchTermIdxWriterRespV2
	//hashedTermToResp map[string]*SearchTermIdxWriterRespV2
	Error error
}

type SearchTermCollectingRespV2 struct {
	respMap       map[*SearchTermIdxWriterV2]error
	processedDocs map[string]map[string]map[string]uint64 // hashedTerm -> (term -> (docID -> ver))
	Error         error
}

// read one block from each source(document)
// hashedTerm -> source blocks (candidate)
func (batch *SearchTermBatchV2) processStiBatchParaReading(readToWriteChan chan<- *SearchTermReadingRespV2) {

	defer close(readToWriteChan)

	for {
		// hashedTerm -> sources(doc1Block2, doc2Block2, doc3Block2)
		hashedTermWriterToBlks := make(map[*SearchTermIdxWriterV2][]*SearchTermIdxSourceBlockV2)

		blkToChan := make(map[SearchTermIdxSourceV2](chan *SearchTermBlockRespV2))

		// read one block from each source(document)
		for _, source := range batch.SourceList {

			blkChan := make(chan *SearchTermBlockRespV2)
			blkToChan[source] = blkChan

			go func(source SearchTermIdxSourceV2, batch *SearchTermBatchV2, blkChan chan<- *SearchTermBlockRespV2) {
				defer close(blkChan)
				blk, err := source.GetNextSourceBlock(batch.termList)
				blkChan <- &SearchTermBlockRespV2{blk, err}
			}(source, batch, blkChan)
		}

		for source, blkChan := range blkToChan {
			blkResp := <-blkChan
			err := blkResp.Error
			blk := blkResp.Block

			if err != nil && err != io.EOF {
				readToWriteChan <- &SearchTermReadingRespV2{hashedTermWriterToBlks, err}
				return
			}

			if blk != nil && len(blk.TermOffset) > 0 {
				for _, hashedTermWriter := range batch.sourceToHashedTermWriters[source] {
					//TODO check if blk contains hashedTerm
					blkList := hashedTermWriterToBlks[hashedTermWriter]
					if blkList == nil {
						blkList = make([]*SearchTermIdxSourceBlockV2, 0, 1)
					}
					hashedTermWriterToBlks[hashedTermWriter] = append(blkList, blk)
				}
			}
		}

		// Finish reading all blocks
		if len(hashedTermWriterToBlks) == 0 {
			readToWriteChan <- &SearchTermReadingRespV2{hashedTermWriterToBlks, nil}
			return
		}

		// Finish reading one more batch, not finish all yet, just go on with the loop
		readToWriteChan <- &SearchTermReadingRespV2{hashedTermWriterToBlks, nil}
	}

}

func (batch *SearchTermBatchV2) processStiBatchParaWriting(
	readToWriteChan <-chan *SearchTermReadingRespV2,
	writeToCollectChan chan<- *SearchTermWritingRespV2) {
	defer close(writeToCollectChan)

	for {
		readResp, more := <-readToWriteChan

		if more {

			// Parse the response from reading
			hashedTermToBlks := readResp.hashedTermWriterToBlks
			err := readResp.Error

			// Check errors
			if err != nil {
				writeToCollectChan <- &SearchTermWritingRespV2{nil, err}
				return
			}

			// Process Source Block in Parallel
			stiwToChan := make(map[*SearchTermIdxWriterV2]chan *SearchTermIdxWriterRespV2)
			for hashedTermWriter, blkList := range hashedTermToBlks {
				if len(blkList) > 0 {
					hashedTermChan := make(chan *SearchTermIdxWriterRespV2)
					stiwToChan[hashedTermWriter] = hashedTermChan

					// process each hashedTerm(writer) in a separate goroutine
					go func(terms []*SearchTermIdxTermInfo,
						writer *SearchTermIdxWriterV2,
						blkList []*SearchTermIdxSourceBlockV2,
						stiwChan chan<- *SearchTermIdxWriterRespV2) {
						defer close(stiwChan)

						stibs, processedDocs, err := writer.ProcessSourceBlocks(blkList)

						if err != nil && err != io.EOF {
							stiwChan <- &SearchTermIdxWriterRespV2{stibs, processedDocs, err}
						} else {
							stiwChan <- &SearchTermIdxWriterRespV2{stibs, processedDocs, nil}
						}
					}(batch.HashedTermToTerms[hashedTermWriter.HashedTerm],
						hashedTermWriter,
						blkList,
						hashedTermChan)

				}
			}

			// Wait for each parallel writing to finish
			respMap := make(map[*SearchTermIdxWriterV2]*SearchTermIdxWriterRespV2)
			for hashedTerm, hashedTermChan := range stiwToChan {
				respMap[hashedTerm] = <-hashedTermChan
			}

			writeToCollectChan <- &SearchTermWritingRespV2{respMap, nil}

		} else {
			return
		}

	}
}

// TODO: optimize by parallelism
// process one hashedTerm writer
func (stiw *SearchTermIdxWriterV2) ProcessSourceBlocks(sourceBlocks []*SearchTermIdxSourceBlockV2) (
	[]*SearchTermIdxBlkV2, // returned block
	map[string]map[string]uint64, // term -> (docID -> docVer)
	error) {

	var err error
	var returnBlocks []*SearchTermIdxBlkV2
	processedDocs := make(map[string]map[string]uint64)

	cloneSourceBlocks := make([]*SearchTermIdxSourceBlockV2, 0, len(sourceBlocks))

	// Remove all sources with lower version than highest document version
	for _, blk := range sourceBlocks {
		highVer, exist := stiw.highDocVerMap[blk.DocID]
		//if exist && highVer <= blk.DocVer && !stiw.delDocMap[blk.DocID] {
		if exist && highVer <= blk.DocVer {
			cloneSourceBlocks = append(cloneSourceBlocks, blk)
		}
	}

	// Read the old STI block
	if stiw.oldStiBlk == nil {
		err = stiw.getOldSearchTermIndexBlock()
		if err != nil && err != io.EOF {
			return nil, nil, err
		}
	}

	// Create the new STI block to be written to
	if stiw.newStiBlk == nil {
		stiw.newStiBlk = CreateSearchTermIdxBlkV2(stiw.newSti.GetMaxBlockDataSize())
	}

	// term -> blocks
	srcOffsetMap := make(map[*SearchTermIdxSourceBlockV2]map[string][]uint64)
	for _, srcBlk := range cloneSourceBlocks {
		for _, term := range stiw.Terms {
			if _, ok := srcBlk.TermOffset[term]; ok {
				if _, ok := srcOffsetMap[srcBlk]; !ok {
					srcOffsetMap[srcBlk] = make(map[string][]uint64)
				}
				srcOffsetMap[srcBlk][term] = append(srcOffsetMap[srcBlk][term], srcBlk.TermOffset[term]...)
			}
		}
	}

	// Process incoming source blocks until all are empty
	for sourceNotEmpty := true; sourceNotEmpty; {

		// TODO: optimization parallel process terms
		var blks []*SearchTermIdxBlkV2 = nil
		var processed map[string]map[string]uint64 // term -> (docID -> docVer)
		sourceNotEmpty, processed, blks, err = stiw.processSourceBlocks(cloneSourceBlocks, srcOffsetMap, copyLen)
		if err != nil {
			return nil, nil, err
		}
		for term, docIDVer := range processed {
			for id, ver := range docIDVer {
				if _, ok := processedDocs[term]; !ok {
					processedDocs[term] = make(map[string]uint64)
				}
				processedDocs[term][id] = ver

			}
		}

		returnBlocks = append(returnBlocks, blks...)

		blks, processed, err = stiw.processOldStiBlock(copyLen, false)
		if err != nil && err != io.EOF {
			return nil, nil, err
		}
		for id, ver := range processed {
			processedDocs[id] = ver
		}
		returnBlocks = append(returnBlocks, blks...)
	}

	return returnBlocks, processedDocs, err
}

func (stiw *SearchTermIdxWriterV2) processOldStiBlock(copyLen int, writeFull bool) (
	[]*SearchTermIdxBlkV2, map[string]map[string]uint64, error) {
	var err error = nil
	returnBlocks := make([]*SearchTermIdxBlkV2, 0, 10)
	processedDocs := make(map[string]map[string]uint64)
	if stiw.oldStiBlk == nil {
		// No more old STI blocks to process
		return returnBlocks, processedDocs, io.EOF
	}

	// Process the old STI block
	for i := 0; i < len(stiw.oldStiBlkDocIDs); i++ {
		docID := stiw.oldStiBlkDocIDs[i]

		oldDocUnfinished := false
		newBlockISFull := false

		for term, docVerOffset := range stiw.oldStiBlk.TermDocVerOffset {
			verOffset := docVerOffset[docID]
			if verOffset == nil || len(verOffset.Offsets) == 0 {
				continue
			}

			// Write result to newSTI
			count := utils.Min(len(verOffset.Offsets), copyLen)
			offsets := verOffset.Offsets[:count]
			err = stiw.newStiBlk.AddDocOffsets(term, docID, verOffset.Version, offsets)
			if err != nil {
				if writeFull {
					err = stiw.newSti.WriteNextBlock(stiw.newStiBlk)
					if err != nil {
						return returnBlocks, processedDocs, err
					}

					returnBlocks = append(returnBlocks, stiw.newStiBlk)

					stiw.newStiBlk = CreateSearchTermIdxBlkV2(stiw.newSti.GetMaxBlockDataSize())
					err = stiw.newStiBlk.AddDocOffsets(term, docID, verOffset.Version, offsets)
					if err != nil {
						return returnBlocks, processedDocs, err
					}
				} else {
					// The new block is full. Don't bother processing anymore old STI block
					newBlockISFull = true
					break
				}
			}

			// update processedDocs
			if _, ok := processedDocs[term]; !ok {
				processedDocs[term] = make(map[string]uint64)
			}
			processedDocs[term][docID] = verOffset.Version

			// Remove the successfully processed offsets from the old STI block (in-place change)
			verOffset.Offsets = verOffset.Offsets[count:]
			if len(verOffset.Offsets) > 0 {
				oldDocUnfinished = true
			}
		}

		if newBlockISFull {
			break
		}
		if !oldDocUnfinished { // oldBlock[anyTerm][docID] has been processed
			stiw.oldStiBlkDocIDs = removeString(stiw.oldStiBlkDocIDs, i)
			i--
		}
	}

	if len(stiw.oldStiBlkDocIDs) == 0 { // all docIDs are processed, get next old block
		err = stiw.getOldSearchTermIndexBlock()
		if err != nil && err != io.EOF {
			return returnBlocks, processedDocs, err
		}
	}

	return returnBlocks, processedDocs, err
}

func (stiw *SearchTermIdxWriterV2) processSourceBlocks(
	sourceBlocks []*SearchTermIdxSourceBlockV2,
	srcOffsetMap map[*SearchTermIdxSourceBlockV2]map[string][]uint64,
	copyLen int) (bool, map[string]map[string]uint64, []*SearchTermIdxBlkV2, error) {

	sourceNotEmpty := false
	var err error = nil
	returnBlocks := make([]*SearchTermIdxBlkV2, 0, 10)
	processedDocs := make(map[string]map[string]uint64)

	for _, srcBlk := range sourceBlocks {
		termSrcOffsets := srcOffsetMap[srcBlk]
		for term, srcOffsets := range termSrcOffsets {
			if len(srcOffsets) == 0 {
				continue
			}
			sourceNotEmpty = true
			count := utils.Min(len(srcOffsets), copyLen)
			offsets := srcOffsets[:count]
			err = stiw.newStiBlk.AddDocOffsets(term, srcBlk.DocID, srcBlk.DocVer, offsets)

			// Current block is full. Send it
			if err != nil {
				err = stiw.newSti.WriteNextBlock(stiw.newStiBlk)
				if err != nil {
					return sourceNotEmpty, processedDocs, returnBlocks, err
				}

				returnBlocks = append(returnBlocks, stiw.newStiBlk)

				stiw.newStiBlk = CreateSearchTermIdxBlkV2(stiw.newSti.GetMaxBlockDataSize())
				err = stiw.newStiBlk.AddDocOffsets(term, srcBlk.DocID, srcBlk.DocVer, offsets)
				if err != nil {
					return sourceNotEmpty, processedDocs, returnBlocks, err
				}
			}
			if _, ok := processedDocs[term]; !ok {
				processedDocs[term] = make(map[string]uint64)
			}
			processedDocs[term][srcBlk.DocID] = srcBlk.DocVer
			srcOffsetMap[srcBlk][term] = srcOffsets[count:]
		}
	}

	return sourceNotEmpty, processedDocs, returnBlocks, nil
}

// read one block from old search term index
// remove documents that are outdated or in source_block_delete_list or in global_delete_list
// set stiw.OldStiBlock to the original block read from STI
// set stiw.oldStiBlkDocIDs to valid docIDs
func (stiw *SearchTermIdxWriterV2) getOldSearchTermIndexBlock() error {
	var err error = nil

	if stiw.oldSti != nil {
		// 1. Remove all document versions lower than high version
		// 2. Remove all documents in source block delete list
		// 3. Remove all documents in global delete list
		// 4. Set stiw.oldStiBlock
		// 5. Set stiw.oldStiBlkDocIDs

		stiw.oldStiBlk = nil
		stiw.oldStiBlkDocIDs = make([]string, 0)

		oldSti := stiw.oldSti.(*SearchTermIdxV2)
		stiw.oldStiBlk, err = oldSti.ReadNextBlock()
		if err != nil && err != io.EOF {
			return err
		}
		if stiw.oldStiBlk != nil {
			var blkDocIDs []string

			for term, docIDToVerOffset := range stiw.oldStiBlk.TermDocVerOffset {
				for docID, docVerOff := range docIDToVerOffset {
					highVer, exist := stiw.highDocVerMap[docID]
					if (exist && highVer > docVerOff.Version) || stiw.termDelDocMap[term][docID] {
						delete(stiw.oldStiBlk.TermDocVerOffset[term], docID)
					} else {
						blkDocIDs = append(blkDocIDs, docID)
					}
				}
			}

			stiw.oldStiBlkDocIDs = blkDocIDs
		}
	}

	return err
}

func (batch *SearchTermBatchV2) processStiBatchParaCollecting(writeToCollectChan <-chan *SearchTermWritingRespV2,
	collectToAllChan chan<- *SearchTermCollectingRespV2) {

	respMap := make(map[*SearchTermIdxWriterV2]error)
	processedDocs := make(map[string]map[string]map[string]uint64) // hashedTerm -> (term -> (docID -> docVer))
	defer close(collectToAllChan)

	for {
		writingResp, more := <-writeToCollectChan
		if more {
			stiBlocksResp := writingResp.hashedTermWriterToResp
			err := writingResp.Error

			if err != nil && err != io.EOF {
				collectToAllChan <- &SearchTermCollectingRespV2{respMap, processedDocs, err}
				return
			}

			if len(stiBlocksResp) > 0 {
				for stiw, resp := range stiBlocksResp {
					if resp != nil {
						// If there is already an error for this STIW, do not overwrite
						if respMap[stiw] == nil {
							respMap[stiw] = resp.Error
						}

						// update processed docs
						if resp.ProcessedDocs != nil && len(resp.ProcessedDocs) > 0 {
							if _, ok := processedDocs[stiw.HashedTerm]; !ok {
								processedDocs[stiw.HashedTerm] = make(map[string]map[string]uint64)
							}

							for term, docIdVer := range resp.ProcessedDocs {
								if _, ok := processedDocs[stiw.HashedTerm][term]; !ok {
									processedDocs[stiw.HashedTerm][term] = make(map[string]uint64)
								}
								for id, ver := range docIdVer {
									processedDocs[stiw.HashedTerm][term][id] = ver
								}
							}

						}
					}
				}
			}
		} else {
			collectToAllChan <- &SearchTermCollectingRespV2{respMap, processedDocs, nil}
			return
		}

	}
}

// process search term index
func (batch *SearchTermBatchV2) processStiAll(event *utils.TimeEvent) (
	map[string]map[string]map[string]uint64, // hashedTerm -> ( term -> (docID -> docVer))
	map[*SearchTermIdxWriterV2]error, // hashedTerm writer -> error
	error) { // some other error
	e1 := utils.AddSubEvent(event, "resetSources")
	for _, source := range batch.SourceList {
		source.Reset()
	}
	utils.EndEvent(e1)

	e2 := utils.AddSubEvent(event, "processStiBatches in 3 threads")

	// Prepare Channels for reading -> writing -> stiAll
	readToWriteChan := make(chan *SearchTermReadingRespV2)
	writeToCollectChan := make(chan *SearchTermWritingRespV2)
	collectToAllChan := make(chan *SearchTermCollectingRespV2)

	// Start to wait for finished writing block, once received, begin to collect and merge the result
	go batch.processStiBatchParaCollecting(writeToCollectChan, collectToAllChan)

	// Start to wait for finished reading block, once received, begin to write
	go batch.processStiBatchParaWriting(readToWriteChan, writeToCollectChan)

	// Start parallel reading, once finished, send each block to writing
	go batch.processStiBatchParaReading(readToWriteChan)

	// wait for collecting to finish
	collectResp := <-collectToAllChan
	respMap := collectResp.respMap
	err := collectResp.Error
	processedDocs := collectResp.processedDocs

	if err != nil && err != io.EOF {
		return nil, nil, err
	}

	utils.EndEvent(e2)

	e3 := utils.AddSubEvent(event, "closeStiw")

	type closeStiwResp struct {
		processedDocs map[string]map[string]uint64
		err           error
	}

	stiwToChan := make(map[*SearchTermIdxWriterV2](chan closeStiwResp))
	for _, stiw := range batch.HashedTermToWriter {
		stiwChan := make(chan closeStiwResp)
		stiwToChan[stiw] = stiwChan
		go func(stiw *SearchTermIdxWriterV2, stiwChan chan closeStiwResp) {
			defer close(stiwChan)
			_, docs, err := stiw.Close()
			stiwChan <- closeStiwResp{docs, err}
		}(stiw, stiwChan)
	}

	for stiw, stiwChan := range stiwToChan {
		resp := <-stiwChan
		err := resp.err
		if err != nil {
			_, ok := respMap[stiw]
			if !ok {
				respMap[stiw] = err
			}
		} else {
			// update processed docs
			if resp.processedDocs != nil && len(resp.processedDocs) > 0 {

				if _, ok := processedDocs[stiw.HashedTerm]; !ok {
					processedDocs[stiw.HashedTerm] = make(map[string]map[string]uint64)
				}
				for term, docIdVer := range resp.processedDocs {
					if _, ok := processedDocs[stiw.HashedTerm][term]; !ok {
						processedDocs[stiw.HashedTerm][term] = make(map[string]uint64)
					}
					for id, ver := range docIdVer {
						processedDocs[stiw.HashedTerm][term][id] = ver
					}
				}

			}
		}
	}
	utils.EndEvent(e3)
	return processedDocs, respMap, nil
}

//////////////////////////////////////////////////////////////////
//
//                 Search HashedTerm Batch Writer V2
//
//////////////////////////////////////////////////////////////////

type SearchTermIdxWriterRespV2 struct {
	Blocks        []*SearchTermIdxBlkV2
	ProcessedDocs map[string]map[string]uint64 // term -> (docID -> docVer)
	Error         error
}

type SearchTermIdxWriterV2 struct {
	HashedTerm string   // hashed term
	Terms      []string // plain terms
	Owner      common.SearchIdxOwner
	TermKey    *sscrypto.StrongSaltKey
	IndexKey   *sscrypto.StrongSaltKey
	newSti     *SearchTermIdxV2
	oldSti     common.SearchTermIdx
	newStiBlk  *SearchTermIdxBlkV2
	oldStiBlk  *SearchTermIdxBlkV2

	// TODO: check if necessary
	updateID        string
	termDelDocMap   map[string]map[string]bool // term -> delDocs
	highDocVerMap   map[string]uint64          // DocID -> DocVer
	oldStiBlkDocIDs []string                   //TODO term->oldStiBlkDocIDs
}

func CreateSearchTermIdxWriterV2(sdc client.StrongDocClient, owner common.SearchIdxOwner,
	hashedTerm string, termSources []*SearchTermBatchSources,
	termDelDocMap map[string]map[string]bool,
	termKey, indexKey *sscrypto.StrongSaltKey) (*SearchTermIdxWriterV2, error) {

	var err error

	stiw := &SearchTermIdxWriterV2{
		Owner:      owner,
		HashedTerm: hashedTerm,
		TermKey:    termKey,
		IndexKey:   indexKey,
		newSti:     nil,
		oldSti:     nil,
		newStiBlk:  nil,
		oldStiBlk:  nil,

		updateID:        "",
		highDocVerMap:   make(map[string]uint64),
		oldStiBlkDocIDs: make([]string, 0),
		termDelDocMap:   termDelDocMap,
	}

	// Record the DocIDs to be deleted
	var allTerms []string
	for _, termSource := range termSources {
		allTerms = append(allTerms, termSource.Term)
		// Find the highest version of each DocID. This is so we can remove
		// any DocID with lower version
		addSource := termSource.AddSources
		for _, stis := range addSource {
			if ver, exist := stiw.highDocVerMap[stis.GetDocID()]; !exist || ver < stis.GetDocVer() {
				stiw.highDocVerMap[stis.GetDocID()] = stis.GetDocVer()
			}
		}
		delSource := termSource.DelSources
		for _, stis := range delSource {
			if ver, exist := stiw.highDocVerMap[stis.GetDocID()]; !exist || ver < stis.GetDocVer() {
				stiw.highDocVerMap[stis.GetDocID()] = stis.GetDocVer()
			}
		}

	}
	stiw.Terms = allTerms

	// Open previous STI if there is one
	updateID, _ := GetLatestUpdateIDV2(sdc, owner, hashedTerm, termKey) // updateID of old STI
	if updateID != "" {
		stiw.oldSti, err = OpenSearchTermIdxV2(sdc, owner, hashedTerm, termKey, indexKey, updateID)
		if err != nil {
			stiw.oldSti = nil
		}
		// update stiw.highDocVerMap
		err = stiw.updateHighDocVersion(sdc, owner, hashedTerm, termKey, indexKey, updateID)
		if err != nil {
			return nil, err
		}
	}

	// Create new STI
	stiw.newSti, err = CreateSearchTermIdxV2(sdc, owner, hashedTerm, termKey, indexKey, nil, nil)
	if err != nil {
		return nil, err
	}

	stiw.updateID = stiw.newSti.updateID
	return stiw, nil
}

// check document highest versions
// read from sortedDocIndex if exists
// else read from old search term index if exists
func (stiw *SearchTermIdxWriterV2) updateHighDocVersion(sdc client.StrongDocClient, owner common.SearchIdxOwner, hashedTerm string,
	termKey, indexKey *sscrypto.StrongSaltKey, updateID string) error {

	ssdi, err := OpenSearchSortDocIdxV2(sdc, owner, hashedTerm, termKey, indexKey, updateID)
	if err == nil {
		err = nil
		for err == nil {
			var blk *SearchSortDocIdxBlkV2 = nil
			blk, err = ssdi.ReadNextBlock()
			if err != nil && err != io.EOF {
				return err
			}
			if blk != nil {
				for _, docIDVer := range blk.termToDocIDVer {
					for docID, docVer := range docIDVer {
						if highVer, exist := stiw.highDocVerMap[docID]; exist && highVer < docVer {
							stiw.highDocVerMap[docID] = docVer
						}
					}

				}
			}
		}
		ssdi.Close()
	} else {
		if stiw.oldSti != nil {
			sti := stiw.oldSti.(*SearchTermIdxV2)
			var err error
			for err == nil {
				var blk *SearchTermIdxBlkV2 = nil
				blk, err = sti.ReadNextBlock()
				if err != nil && err != io.EOF {
					return err
				}
				if blk != nil {
					for _, docIDVer := range blk.TermDocVerOffset {
						for docID, docVerOff := range docIDVer {
							if highVer, exist := stiw.highDocVerMap[docID]; exist && highVer < docVerOff.Version {
								stiw.highDocVerMap[docID] = docVerOff.Version
							}
						}

					}
				}
			}
			err = sti.Reset()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (stiw *SearchTermIdxWriterV2) Close() ([]*SearchTermIdxBlkV2, map[string]map[string]uint64, error) {
	returnBlocks := make([]*SearchTermIdxBlkV2, 0, 10)
	var blks []*SearchTermIdxBlkV2 = nil
	var err, finalErr error = nil, nil
	var processedDocs map[string]map[string]uint64

	if stiw.oldStiBlk != nil {
		for err != nil {
			blks, processedDocs, err = stiw.processOldStiBlock(copyLen, true)
			if err != nil && err != io.EOF {
				finalErr = err
			}
			if len(blks) > 0 {
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

	return returnBlocks, processedDocs, finalErr
}

func (stiw *SearchTermIdxWriterV2) GetUpdateID() string {
	return stiw.updateID
}

func (batch *SearchTermBatchV2) IsEmpty() bool {
	return (len(batch.HashedTermToWriter) == 0)
}
