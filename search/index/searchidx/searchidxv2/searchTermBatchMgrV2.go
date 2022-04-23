package searchidxv2

import (
	"io"
	"sort"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/tokenizer"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

const copyLen = 10

func removeString(stringList []string, i int) []string {
	if stringList != nil && i >= 0 && i < len(stringList) {
		stringList[i] = stringList[len(stringList)-1]
		stringList = stringList[:len(stringList)-1]
	}
	return stringList
}

//////////////////////////////////////////////////////////////////
//
//               Search Term Batch Manager V2
//
//////////////////////////////////////////////////////////////////

type SearchTermBatchMgrV2 struct {
	Owner            common.SearchIdxOwner
	SearchIdxSources []SearchTermIdxSourceV2
	TermKey          *sscrypto.StrongSaltKey
	IndexKey         *sscrypto.StrongSaltKey
	BatchSize        uint32
	BucketCount      uint32
	TokenizerType    tokenizer.TokenizerType
	delDocs          *DeletedDocsV2 // global delete list
	termBucketList   []*SearchTermBucket
}

type SearchTermBucket struct {
	TermID      string
	TermKey     *sscrypto.StrongSaltKey
	IndexKey    *sscrypto.StrongSaltKey
	TermSources []*SearchTermSources
}

type SearchTermSources struct {
	Term       string // plain term
	AddSources []SearchTermIdxSourceV2
	DelSources []SearchTermIdxSourceV2
	delDocs    *DeletedDocsV2 // global delete list
}

func CreateSearchTermBatchMgrV2(owner common.SearchIdxOwner, sources []SearchTermIdxSourceV2,
	termKey, indexKey *sscrypto.StrongSaltKey, batchSize uint32, bucketCount uint32,
	delDocs *DeletedDocsV2) (*SearchTermBatchMgrV2, error) {

	mgr := &SearchTermBatchMgrV2{
		Owner:          owner,
		TermKey:        termKey,
		IndexKey:       indexKey,
		BatchSize:      batchSize,
		BucketCount:    bucketCount,
		TokenizerType:  getTokenizerType(sources),
		delDocs:        delDocs,
		termBucketList: make([]*SearchTermBucket, 0, 100)}

	termSourcesMap := make(map[string]*SearchTermSources) // term -> termSources
	for _, source := range sources {
		for _, term := range source.GetAllTerms() {
			termSources := termSourcesMap[term]
			if termSources == nil {
				termSources = &SearchTermSources{
					Term:       term,
					AddSources: make([]SearchTermIdxSourceV2, 0, 10),
					DelSources: make([]SearchTermIdxSourceV2, 0, 10),
					delDocs:    delDocs}
				termSourcesMap[term] = termSources
			}
			termSources.AddSources = append(termSources.AddSources, source)
		}

		for _, term := range source.GetDelTerms() {
			termSources := termSourcesMap[term]
			if termSources == nil {
				termSources = &SearchTermSources{
					Term:       term,
					AddSources: make([]SearchTermIdxSourceV2, 0, 10),
					DelSources: make([]SearchTermIdxSourceV2, 0, 10),
					delDocs:    delDocs}
				termSourcesMap[term] = termSources
			}
			termSources.DelSources = append(termSources.DelSources, source)
		}
	}

	termIDMap := make(map[string]*SearchTermBucket) // termID -> SearchTermBucket
	for term, termSources := range termSourcesMap {
		termID, err := common.GetTermID(term, termKey, bucketCount, common.STI_V2)
		if err != nil {
			return nil, err
		}

		termBucket := termIDMap[termID]
		if termBucket == nil {
			termBucket = &SearchTermBucket{
				TermID:   termID,
				IndexKey: indexKey,
				TermKey:  termKey,
			}
			termIDMap[termID] = termBucket
			mgr.termBucketList = append(mgr.termBucketList, termBucket)
		}
		termBucket.TermSources = append(termBucket.TermSources, termSources)
	}
	return mgr, nil
}

func getTokenizerType(sources []SearchTermIdxSourceV2) (tokenType tokenizer.TokenizerType) {
	for _, source := range sources {
		tokenType = source.GetTokenizerType()
	}
	return
}

//////////////////////////////////////////////////////////////////
//
//                    Search Term Batch V2
//
//////////////////////////////////////////////////////////////////
type searchTermIdxTermInfo struct {
	term       string // plain term
	addSources []SearchTermIdxSourceV2
	delSources []SearchTermIdxSourceV2
	delDocMap  map[string]bool //
}

type SearchTermBatchV2 struct {
	Owner           common.SearchIdxOwner                              // owner
	SourceList      []SearchTermIdxSourceV2                            // all docs
	TokenizerType   tokenizer.TokenizerType                            // tokenizer type of the sources
	termList        []string                                           // all terms
	termIDLIst      []string                                           // all termIDs
	termIDToWriter  map[string]*SearchTermIdxWriterV2                  // termID -> index writer
	termIDToTerms   map[string][]*searchTermIdxTermInfo                // termID -> terms mapped to this bucket
	sourceToTermIDs map[SearchTermIdxSourceV2][]string                 // source -> []termID
	sourceToStiw    map[SearchTermIdxSourceV2][]*SearchTermIdxWriterV2 // source -> search term index writers
	processedDocs   map[string]map[string]map[string]uint64            // termID -> (term -> (docID -> docVer))
}

func (mgr *SearchTermBatchMgrV2) GetNextTermBatch(sdc client.StrongDocClient) (*SearchTermBatchV2, error) {
	batchSize := utils.Min(len(mgr.termBucketList), int(mgr.BatchSize))

	// The term bucket batch to process
	termBuckets := mgr.termBucketList[:batchSize]
	mgr.termBucketList = mgr.termBucketList[batchSize:]

	return CreateSearchTermBatchV2(sdc, mgr.Owner, mgr.TokenizerType, termBuckets)
}

// TODO: optimization
func CreateSearchTermBatchV2(sdc client.StrongDocClient, owner common.SearchIdxOwner,
	tokenizerType tokenizer.TokenizerType, termBucketBatch []*SearchTermBucket) (*SearchTermBatchV2, error) {
	batch := &SearchTermBatchV2{
		Owner:           owner,
		SourceList:      nil,                                     // all docs
		TokenizerType:   tokenizerType,                           // tokenizer
		termIDToWriter:  make(map[string]*SearchTermIdxWriterV2), // termID -> writer
		termIDToTerms:   nil,
		sourceToTermIDs: nil,
		termList:        nil, // all terms
		termIDLIst:      nil, // all bucket IDs
		processedDocs:   make(map[string]map[string]map[string]uint64),
	}

	// TermSource -> (TermID -> bool)
	sourceToTermIDsMap := make(map[SearchTermIdxSourceV2]map[string]bool)
	for _, termBucket := range termBucketBatch {
		termID := termBucket.TermID
		for _, term := range termBucket.TermSources {
			for _, source := range term.AddSources {
				if _, ok := sourceToTermIDsMap[source]; !ok {
					sourceToTermIDsMap[source] = make(map[string]bool)
				}
				sourceToTermIDsMap[source][termID] = true
			}
		}
	}

	// TermSource -> []TermID
	sourceToTermIDsList := make(map[SearchTermIdxSourceV2][]string)
	for source, termIDMap := range sourceToTermIDsMap {
		var termIDList []string
		if len(termIDMap) > 0 {
			for termID := range termIDMap {
				termIDList = append(termIDList, termID)
			}
		}
		sourceToTermIDsList[source] = termIDList
	}
	batch.sourceToTermIDs = sourceToTermIDsList

	// sourceList
	batch.SourceList = make([]SearchTermIdxSourceV2, 0, len(batch.sourceToTermIDs))
	for stis := range batch.sourceToTermIDs {
		batch.SourceList = append(batch.SourceList, stis)
	}

	// TermID -> []SearchTermIdxTermInfo
	termIDToTerms := make(map[string][]*searchTermIdxTermInfo)
	// TermID -> (Term -> (DocID -> bool))
	termIDToAllDelDocsMap := make(map[string]map[string]map[string]bool)
	for _, termBucket := range termBucketBatch {
		var terms []*searchTermIdxTermInfo
		termIDToAllDelDocsMap[termBucket.TermID] = make(map[string]map[string]bool)
		for _, termSource := range termBucket.TermSources {
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

			termInfo := &searchTermIdxTermInfo{
				term:       termSource.Term,
				addSources: termSource.AddSources,
				delSources: termSource.DelSources,
				delDocMap:  delDocMap,
			}
			terms = append(terms, termInfo)
			termIDToAllDelDocsMap[termBucket.TermID][termSource.Term] = delDocMap
		}
		termIDToTerms[termBucket.TermID] = terms
	}
	batch.termIDToTerms = termIDToTerms

	// TermIDToWriter
	var allTerms []string
	var allTermIDs []string
	for _, termBucket := range termBucketBatch {
		allTermIDs = append(allTermIDs, termBucket.TermID)
		// create term index writer for every termID
		writer, err := CreateSearchTermIdxWriterV2(sdc, owner, tokenizerType,
			termBucket.TermID, termBucket.TermSources,
			termIDToAllDelDocsMap[termBucket.TermID],
			termBucket.TermKey, termBucket.IndexKey)
		if err != nil {
			return nil, err
		}
		// collect all terms
		for _, termSource := range termBucket.TermSources {
			allTerms = append(allTerms, termSource.Term)

		}
		batch.termIDToWriter[termBucket.TermID] = writer
	}

	sort.Strings(allTerms)
	batch.termList = allTerms
	sort.Strings(allTermIDs)
	batch.termIDLIst = allTermIDs

	sourceToStiws := make(map[SearchTermIdxSourceV2][]*SearchTermIdxWriterV2)
	for source, termIDMap := range sourceToTermIDsMap {
		if len(termIDMap) > 0 {
			for termID := range termIDMap {
				stiw := batch.termIDToWriter[termID]
				sourceToStiws[source] = append(sourceToStiws[source], stiw)
			}
		}
	}
	batch.sourceToStiw = sourceToStiws

	return batch, nil
}

func (batch *SearchTermBatchV2) ProcessTermBatch(sdc client.StrongDocClient, event *utils.TimeEvent) (map[string]error, error) {
	respMap := make(map[string]error)

	e1 := utils.AddSubEvent(event, "processStiAll")
	stiResp, err := batch.processStiAll(e1)
	if err != nil {
		return respMap, err
	}
	utils.EndEvent(e1)

	stiSuccess := make([]*SearchTermIdxWriterV2, 0, len(stiResp))
	for stiw, err := range stiResp {
		respMap[stiw.TermID] = utils.FirstError(respMap[stiw.TermID], err)
		if err == nil {
			stiSuccess = append(stiSuccess, stiw)
		}
	}

	e2 := utils.AddSubEvent(event, "processSsdiAll")
	ssdiResp, err := batch.processSsdiAll(sdc, stiSuccess)
	if err != nil {
		return respMap, err
	}

	for stiw, err := range ssdiResp {
		respMap[stiw.TermID] = utils.FirstError(respMap[stiw.TermID], err)
	}
	utils.EndEvent(e2)

	return respMap, nil
}

// ------------------ Process SSDI------------------
func (batch *SearchTermBatchV2) processSsdiAll(sdc client.StrongDocClient,
	stiwList []*SearchTermIdxWriterV2) (map[*SearchTermIdxWriterV2]error, error) {

	ssdiToChan := make(map[*SearchTermIdxWriterV2]chan error)

	for _, stiw := range stiwList {
		ssdiChan := make(chan error)
		ssdiToChan[stiw] = ssdiChan

		// process each bucket in a separate thread
		go func(stiw *SearchTermIdxWriterV2,
			docs map[string]map[string]uint64, // [term] -> [[docID] -> ver]
			ssdiChan chan<- error) {
			defer close(ssdiChan)
			ssdi, err := CreateSearchSortDocIdxV2(sdc, batch.Owner, stiw.TermID,
				stiw.GetUpdateID(), stiw.TermKey, stiw.IndexKey, batch.TokenizerType, docs)
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
		}(stiw, batch.processedDocs[stiw.TermID], ssdiChan)

	}

	respMap := make(map[*SearchTermIdxWriterV2]error)
	for stiw, ssdiChan := range ssdiToChan {
		respMap[stiw] = <-ssdiChan
	}

	return respMap, nil
}

// ------------------ Process STI------------------
type StiReadRespV2 struct {
	stiwToBlks map[*SearchTermIdxWriterV2][]*SearchTermIdxSourceBlockV2
	Error      error
}

type StiWriteRespV2 struct {
	stiwToResp map[*SearchTermIdxWriterV2]*SearchTermIdxWriterRespV2
	Error      error
}

// read one block from each source(document)
// stiw -> source blocks (candidate)
func (batch *SearchTermBatchV2) stiReadThread(readToWriteChan chan<- *StiReadRespV2) {
	type blockRespV2 struct {
		Block *SearchTermIdxSourceBlockV2
		Error error
	}

	defer close(readToWriteChan)

	for {
		// source -> block response
		blkToChan := make(map[SearchTermIdxSourceV2](chan *blockRespV2))

		// read one block from each source(document)
		for _, source := range batch.SourceList {
			blkChan := make(chan *blockRespV2)
			blkToChan[source] = blkChan

			go func(source SearchTermIdxSourceV2, batch *SearchTermBatchV2, blkChan chan<- *blockRespV2) {
				defer close(blkChan)
				blk, err := source.GetNextSourceBlock(batch.termList)
				blkChan <- &blockRespV2{blk, err}
			}(source, batch, blkChan)
		}

		// stiw -> sources(doc1Block2, doc2Block2, doc3Block2)
		stiwToBlks := make(map[*SearchTermIdxWriterV2][]*SearchTermIdxSourceBlockV2)

		for source, blkChan := range blkToChan {
			blkResp := <-blkChan
			err := blkResp.Error
			blk := blkResp.Block

			if err != nil && err != io.EOF { // Unrecoverable error
				readToWriteChan <- &StiReadRespV2{stiwToBlks, err}
				return
			}

			if blk != nil && len(blk.TermOffset) > 0 {
				for _, stiw := range batch.sourceToStiw[source] {
					stiwToBlks[stiw] = append(stiwToBlks[stiw], blk)
				}
			}
		}

		readToWriteChan <- &StiReadRespV2{stiwToBlks, nil}

		// Finish reading all blocks, exit function
		if len(stiwToBlks) == 0 {
			return
		}
	}
}

func (batch *SearchTermBatchV2) stiWriteThread(stiReadChan <-chan *StiReadRespV2, stiWriteChan chan<- *StiWriteRespV2) {

	defer close(stiWriteChan)

	for {
		readResp, more := <-stiReadChan

		// The channel to read from has closed. Nothing else to process for this thread
		if !more {
			return
		}

		// Parse the response from reading
		stiwToBlks := readResp.stiwToBlks
		err := readResp.Error

		// Check errors
		if err != nil {
			stiWriteChan <- &StiWriteRespV2{nil, err}
			return
		}

		// Process Source Block in Parallel
		stiwToChan := make(map[*SearchTermIdxWriterV2]chan *SearchTermIdxWriterRespV2)
		for stiw, blkList := range stiwToBlks {
			if len(blkList) <= 0 {
				continue
			}

			stiwChan := make(chan *SearchTermIdxWriterRespV2)
			stiwToChan[stiw] = stiwChan

			// process each bucket writer in a separate goroutine
			go func(terms []*searchTermIdxTermInfo,
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
			}(batch.termIDToTerms[stiw.TermID],
				stiw,
				blkList,
				stiwChan)
		}

		// Wait for each parallel writing to finish
		respMap := make(map[*SearchTermIdxWriterV2]*SearchTermIdxWriterRespV2)
		for stiw, stiwChan := range stiwToChan {
			respMap[stiw] = <-stiwChan
		}

		stiWriteChan <- &StiWriteRespV2{respMap, nil}
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

type StiCloseRespV2 struct {
	stiwErrorMap map[*SearchTermIdxWriterV2]error
	Error        error
}

func (batch *SearchTermBatchV2) stiCloseThread(writeChan <-chan *StiWriteRespV2, closeChan chan<- *StiCloseRespV2) {
	stiwErrorMap := make(map[*SearchTermIdxWriterV2]error)
	batch.processedDocs = make(map[string]map[string]map[string]uint64) // termID -> (term -> (docID -> docVer))
	defer close(closeChan)

	closeResp := &StiCloseRespV2{stiwErrorMap, nil}

	// Process the write channel
	for {
		writeResp, more := <-writeChan
		if !more {
			break // Write channel closed. Stop processing and proceed forward
		}

		err := writeResp.Error
		if err != nil && err != io.EOF {
			closeResp.Error = err
			break // Write channel error. Stop processing and proceed forward
		}

		stiwToResp := writeResp.stiwToResp
		if len(stiwToResp) <= 0 {
			continue
		}

		for stiw, writeResp := range stiwToResp {
			if writeResp == nil {
				continue
			}

			// If there is already an error for this stiw, do not overwrite
			if stiwErrorMap[stiw] == nil {
				stiwErrorMap[stiw] = writeResp.Error
			}

			// update processed docs
			batch.updateProcessedDocs(stiw.TermID, writeResp.ProcessedDocs)
		}
	} // Processing write channel

	type closeStiwResp struct {
		processedDocs map[string]map[string]uint64 // term -> (docID -> docVer)
		err           error
	}

	// Close all STIW in parallel
	stiwToCloseChan := make(map[*SearchTermIdxWriterV2](chan *closeStiwResp))
	for _, stiw := range batch.termIDToWriter {
		stiwCloseChan := make(chan *closeStiwResp)
		stiwToCloseChan[stiw] = stiwCloseChan
		go func(stiw *SearchTermIdxWriterV2, stiwCloseChan chan *closeStiwResp) {
			defer close(stiwCloseChan)
			_, docs, err := stiw.Close()
			stiwCloseChan <- &closeStiwResp{docs, err}
		}(stiw, stiwCloseChan)
	}

	for stiw, stiwCloseChan := range stiwToCloseChan {
		// If there is already an error for this stiw, do not overwrite
		err := closeResp.Error
		if stiwErrorMap[stiw] == nil {
			stiwErrorMap[stiw] = err
		}

		closeResp := <-stiwCloseChan
		batch.updateProcessedDocs(stiw.TermID, closeResp.processedDocs)
	}

	closeChan <- &StiCloseRespV2{stiwErrorMap, nil}
}

// newProcessedDocs: term -> (docID -> docVer))
func (batch *SearchTermBatchV2) updateProcessedDocs(termID string, newProcessedDocs map[string]map[string]uint64) {
	// update processed docs
	if len(newProcessedDocs) > 0 {
		if _, ok := batch.processedDocs[termID]; !ok {
			batch.processedDocs[termID] = make(map[string]map[string]uint64)
		}

		for term, docIdVer := range newProcessedDocs {
			if _, ok := batch.processedDocs[termID][term]; !ok {
				batch.processedDocs[termID][term] = make(map[string]uint64)
			}
			for id, ver := range docIdVer {
				batch.processedDocs[termID][term][id] = ver
			}
		}
	}
}

// process search term index
func (batch *SearchTermBatchV2) processStiAll(event *utils.TimeEvent) (map[*SearchTermIdxWriterV2]error, error) {
	e1 := utils.AddSubEvent(event, "resetSources")
	for _, source := range batch.SourceList {
		source.Reset()
	}
	utils.EndEvent(e1)

	e2 := utils.AddSubEvent(event, "processStiBatches in 3 threads")

	// Prepare Channels for reading -> writing -> closing
	readChan := make(chan *StiReadRespV2)
	writeChan := make(chan *StiWriteRespV2)
	closeChan := make(chan *StiCloseRespV2)

	// Start parallel reading thread. This thread sends all data read to the writing thread
	go batch.stiReadThread(readChan)

	// Start parallel writing thread. This thread take data from the reading thread and writes them
	go batch.stiWriteThread(readChan, writeChan)

	// Start parallel close thread. This thread collects all results from the writing thread, and
	// closes all the writers when done.
	go batch.stiCloseThread(writeChan, closeChan)

	closeResp := <-closeChan
	respMap := closeResp.stiwErrorMap
	err := closeResp.Error

	utils.EndEvent(e2)

	if err != nil && err != io.EOF {
		return respMap, err
	}

	return respMap, nil
}

//////////////////////////////////////////////////////////////////
//
//               Search Term Bucket Index Writer V2
//
//////////////////////////////////////////////////////////////////

type SearchTermIdxWriterRespV2 struct {
	Blocks        []*SearchTermIdxBlkV2
	ProcessedDocs map[string]map[string]uint64 // term -> (docID -> docVer)
	Error         error
}

type SearchTermIdxWriterV2 struct {
	TermID        string   // term ID
	Terms         []string // plain terms
	Owner         common.SearchIdxOwner
	TermKey       *sscrypto.StrongSaltKey
	IndexKey      *sscrypto.StrongSaltKey
	TokenizerType tokenizer.TokenizerType
	newSti        *SearchTermIdxV2
	oldSti        common.SearchTermIdx
	newStiBlk     *SearchTermIdxBlkV2
	oldStiBlk     *SearchTermIdxBlkV2

	updateID        string
	termDelDocMap   map[string]map[string]bool // term -> delDocs
	highDocVerMap   map[string]uint64          // DocID -> DocVer
	oldStiBlkDocIDs []string
}

func CreateSearchTermIdxWriterV2(sdc client.StrongDocClient,
	owner common.SearchIdxOwner, tokenizerType tokenizer.TokenizerType,
	termID string, termSources []*SearchTermSources,
	termDelDocMap map[string]map[string]bool,
	termKey, indexKey *sscrypto.StrongSaltKey) (*SearchTermIdxWriterV2, error) {

	var err error

	stiw := &SearchTermIdxWriterV2{
		Owner:     owner,
		TermID:    termID,
		TermKey:   termKey,
		IndexKey:  indexKey,
		newSti:    nil,
		oldSti:    nil,
		newStiBlk: nil,
		oldStiBlk: nil,

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
	updateID, _ := GetLatestUpdateIDV2(sdc, owner, termID) // updateID of old STI
	// updateID, _ := GetLatestUpdateIDV2(sdc, owner, termID, termKey) // updateID of old STI
	if updateID != "" {
		stiw.oldSti, err = OpenSearchTermIdxV2(sdc, owner, termID, termKey, indexKey, updateID)
		if err != nil {
			stiw.oldSti = nil
		}
		// update stiw.highDocVerMap
		err = stiw.updateHighDocVersion(sdc, owner, termID, termKey, indexKey, updateID)
		if err != nil {
			return nil, err
		}
	}

	// Create new STI
	stiw.newSti, err = CreateSearchTermIdxV2(sdc, owner, termID, termKey, indexKey, nil, nil, tokenizerType)
	if err != nil {
		return nil, err
	}

	stiw.updateID = stiw.newSti.updateID
	return stiw, nil
}

// check document highest versions
// read from sortedDocIndex if exists
// else read from old search term index if exists
func (stiw *SearchTermIdxWriterV2) updateHighDocVersion(sdc client.StrongDocClient,
	owner common.SearchIdxOwner, termID string,
	termKey, indexKey *sscrypto.StrongSaltKey, updateID string) error {

	ssdi, err := OpenSearchSortDocIdxV2(sdc, owner, termID, termKey, indexKey, updateID)
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
	return (len(batch.termIDToWriter) == 0)
}
