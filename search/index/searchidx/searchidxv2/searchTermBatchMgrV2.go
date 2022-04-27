package searchidxv2

import (
	"io"
	"log"
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

var searchTermIndexAnalyzer tokenizer.Analyzer = nil

func init() {
	analyzer, err := tokenizer.OpenAnalyzer(GetSearchTermIdxAnalyzerType())
	if err != nil {
		log.Fatal(err)
	}
	searchTermIndexAnalyzer = analyzer
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
	globalDelDocs    *DeletedDocsV2 // global delete list
	termIDInfoList   []*SearchTermIDInfo
}

type SearchTermIDInfo struct {
	TermID        string
	Owner         common.SearchIdxOwner
	TermKey       *sscrypto.StrongSaltKey
	IndexKey      *sscrypto.StrongSaltKey
	TokenizerType tokenizer.TokenizerType
	TermInfoList  []*SearchTermInfo
	globalDelDocs *DeletedDocsV2 // global delete list
	writer        *SearchTermIdxWriterV2
}

type SearchTermInfo struct {
	OrigTerm   string // original term
	AnalzdTerm string // analyzed term
	AddSources []SearchTermIdxSourceV2
	DelSources []SearchTermIdxSourceV2
}

type termInfoMap map[string]*SearchTermInfo // origTerm -> termInfo
func (tim termInfoMap) getTermInfo(origTerm string) *SearchTermInfo {
	termInfo := tim[origTerm]
	if termInfo == nil {
		analyzer := GetSearchTermIdxAnalyzer()
		termInfo = &SearchTermInfo{
			OrigTerm:   origTerm,
			AnalzdTerm: analyzer.Analyze(origTerm)[0],
			AddSources: make([]SearchTermIdxSourceV2, 0, 10),
			DelSources: make([]SearchTermIdxSourceV2, 0, 10),
		}
		tim[origTerm] = termInfo
	}

	return termInfo
}

func CreateSearchTermBatchMgrV2(owner common.SearchIdxOwner, sources []SearchTermIdxSourceV2,
	termKey, indexKey *sscrypto.StrongSaltKey, batchSize uint32, bucketCount uint32,
	globalDelDocs *DeletedDocsV2) (*SearchTermBatchMgrV2, error) {

	mgr := &SearchTermBatchMgrV2{
		Owner:          owner,
		TermKey:        termKey,
		IndexKey:       indexKey,
		BatchSize:      batchSize,
		BucketCount:    bucketCount,
		TokenizerType:  GetSearchTermIdxAnalyzerType(),
		globalDelDocs:  globalDelDocs,
		termIDInfoList: make([]*SearchTermIDInfo, 0, 100)}

	if len(globalDelDocs.delDocMap) != len(globalDelDocs.DelDocs) {
		globalDelDocs.delDocMap = make(map[string]bool)
		for _, delDocs := range globalDelDocs.DelDocs {
			globalDelDocs.delDocMap[delDocs] = true
		}
	}

	termInfoMap := make(termInfoMap) // origTerm -> termInfo
	for _, source := range sources {
		for _, addOrigTerm := range source.GetAllTerms() {
			termInfo := termInfoMap.getTermInfo(addOrigTerm)
			termInfo.AddSources = append(termInfo.AddSources, source)
		}

		for _, delOrigTerm := range source.GetDelTerms() {
			termInfo := termInfoMap.getTermInfo(delOrigTerm)
			termInfo.DelSources = append(termInfo.DelSources, source)
		}
	}

	termIDMap := make(map[string]*SearchTermIDInfo) // termID -> SearchTermIDInfo
	for origTerm, termInfo := range termInfoMap {
		termID, err := GetSearchTermID(origTerm, termKey, bucketCount)
		if err != nil {
			return nil, err
		}

		termIDInfo := termIDMap[termID]
		if termIDInfo == nil {
			termIDInfo = &SearchTermIDInfo{
				TermID:        termID,
				Owner:         owner,
				IndexKey:      indexKey,
				TermKey:       termKey,
				TokenizerType: mgr.TokenizerType,
				TermInfoList:  nil,
				globalDelDocs: globalDelDocs,
				writer:        nil,
			}
			termIDMap[termID] = termIDInfo
			mgr.termIDInfoList = append(mgr.termIDInfoList, termIDInfo)
		}
		termIDInfo.TermInfoList = append(termIDInfo.TermInfoList, termInfo)
	}

	return mgr, nil
}

func GetSearchTermIdxAnalyzerType() tokenizer.TokenizerType {
	return tokenizer.TKZER_BLEVE
}

func GetSearchTermIdxAnalyzer() tokenizer.Analyzer {
	return searchTermIndexAnalyzer
}

func GetSearchTermIDs(origTerm string, termKey *sscrypto.StrongSaltKey, bucketCount uint32) ([]string, error) {
	tokens := GetSearchTermIdxAnalyzer().Analyze(origTerm)

	termIDs := make([]string, 0, len(tokens))
	for _, token := range tokens {
		termID, err := common.GetTermID(token, termKey, bucketCount, common.STI_V2)
		if err != nil {
			return nil, err
		}
		termIDs = append(termIDs, termID)
	}

	return termIDs, nil
}

func GetSearchTermID(origTerm string, termKey *sscrypto.StrongSaltKey, bucketCount uint32) (string, error) {
	termIDs, err := GetSearchTermIDs(origTerm, termKey, bucketCount)
	if err != nil {
		return "", err
	}

	if len(termIDs) > 0 {
		return termIDs[0], nil
	}

	return "", nil
}

//////////////////////////////////////////////////////////////////
//
//                    Search Term Batch V2
//
//////////////////////////////////////////////////////////////////

type processedDocs map[string]uint64                      // (docID -> docVer)
type processedTermDocs map[string]processedDocs           // analzdTerm -> processedDocs(docID -> docVer)
type processedTermIDTermDocs map[string]processedTermDocs // termID -> processedTermDocs(term -> (docID -> docVer))

type SearchTermBatchV2 struct {
	Owner           common.SearchIdxOwner              // owner
	SourceList      []SearchTermIdxSourceV2            // all doc sources
	origTermList    []string                           // all original terms
	termIDList      []string                           // all termIDs
	termIDInfoMap   map[string]*SearchTermIDInfo       // termID -> SearchTermIdInfo
	sourceToTermIDs map[SearchTermIdxSourceV2][]string // source -> []termID
	processedDocs   processedTermIDTermDocs            // termID -> (term -> (docID -> docVer))
}

func (mgr *SearchTermBatchMgrV2) GetNextTermBatch(sdc client.StrongDocClient) (*SearchTermBatchV2, error) {
	batchSize := utils.Min(len(mgr.termIDInfoList), int(mgr.BatchSize))

	// The term ID batch to process
	termIDInfoBatch := mgr.termIDInfoList[:batchSize]
	mgr.termIDInfoList = mgr.termIDInfoList[batchSize:]

	return CreateSearchTermBatchV2(sdc, mgr.Owner, termIDInfoBatch)
}

// TODO: optimization
func CreateSearchTermBatchV2(sdc client.StrongDocClient, owner common.SearchIdxOwner,
	termIDInfoBatch []*SearchTermIDInfo) (batch *SearchTermBatchV2, err error) {

	batch = &SearchTermBatchV2{
		Owner:           owner,
		SourceList:      nil,                                      // all sources
		origTermList:    nil,                                      // all terms
		termIDList:      make([]string, 0, len(termIDInfoBatch)),  // all term IDs
		termIDInfoMap:   make(map[string]*SearchTermIDInfo),       // termID -> SearchTermIdInfo
		sourceToTermIDs: make(map[SearchTermIdxSourceV2][]string), // source -> []termID
		processedDocs:   make(processedTermIDTermDocs),
	}

	// Source -> (TermID -> bool)
	sourceToTermIDsMap := make(map[SearchTermIdxSourceV2]map[string]bool)
	for _, termIDInfo := range termIDInfoBatch {
		batch.termIDList = append(batch.termIDList, termIDInfo.TermID)
		batch.termIDInfoMap[termIDInfo.TermID] = termIDInfo

		for _, termInfo := range termIDInfo.TermInfoList {
			batch.origTermList = append(batch.origTermList, termInfo.OrigTerm)

			for _, source := range termInfo.AddSources {
				if _, ok := sourceToTermIDsMap[source]; !ok {
					sourceToTermIDsMap[source] = make(map[string]bool)
				}
				sourceToTermIDsMap[source][termIDInfo.TermID] = true
			}
		}

		termIDInfo.writer, err = CreateSearchTermIdxWriterV2(sdc, termIDInfo)
		if err != nil {
			return nil, err
		}
	}

	sort.Strings(batch.origTermList)
	sort.Strings(batch.termIDList)

	for source, termIDMap := range sourceToTermIDsMap {
		for termID := range termIDMap {
			batch.sourceToTermIDs[source] = append(batch.sourceToTermIDs[source], termID)
		}
		batch.SourceList = append(batch.SourceList, source)
	}

	return batch, nil
}

func (batch *SearchTermBatchV2) ProcessTermBatch(sdc client.StrongDocClient, event *utils.TimeEvent) (
	map[string]error, error) {

	termIDErrMap := make(map[string]error)

	e1 := utils.AddSubEvent(event, "processStiAll")
	stiResp, err := batch.processStiAll(e1)
	if err != nil {
		return termIDErrMap, err
	}
	utils.EndEvent(e1)

	stiSuccess := make([]*SearchTermIdxWriterV2, 0, len(stiResp))
	for stiw, err := range stiResp {
		termID := stiw.TermIDInfo.TermID
		termIDErrMap[termID] = utils.FirstError(termIDErrMap[termID], err)
		if err == nil {
			stiSuccess = append(stiSuccess, stiw)
		}
	}

	e2 := utils.AddSubEvent(event, "processSsdiAll")
	ssdiResp, err := batch.processSsdiAll(sdc, stiSuccess)
	if err != nil {
		return termIDErrMap, err
	}

	for stiw, err := range ssdiResp {
		termID := stiw.TermIDInfo.TermID
		termIDErrMap[termID] = utils.FirstError(termIDErrMap[termID], err)
	}
	utils.EndEvent(e2)

	return termIDErrMap, nil
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
			procTermDocs processedTermDocs, // term -> (docID -> ver)
			ssdiChan chan<- error) {
			defer close(ssdiChan)
			ssdi, err := CreateSearchSortDocIdxV2(sdc, batch.Owner, stiw.TermIDInfo.TermID,
				stiw.GetUpdateID(), stiw.TermIDInfo.TermKey, stiw.TermIDInfo.IndexKey,
				stiw.TermIDInfo.TokenizerType, procTermDocs)
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
		}(stiw, batch.processedDocs[stiw.TermIDInfo.TermID], ssdiChan)
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

			// Process the following in different threads
			go func(source SearchTermIdxSourceV2, batch *SearchTermBatchV2, blkChan chan<- *blockRespV2) {
				defer close(blkChan)
				blk, err := source.GetNextSourceBlock(batch.origTermList)
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
				for _, termID := range batch.sourceToTermIDs[source] {
					stiw := batch.termIDInfoMap[termID].writer
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

func (batch *SearchTermBatchV2) stiWriteThread(stiReadChan <-chan *StiReadRespV2,
	stiWriteChan chan<- *StiWriteRespV2) {

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
			go func(
				stiw *SearchTermIdxWriterV2,
				blkList []*SearchTermIdxSourceBlockV2,
				stiwChan chan<- *SearchTermIdxWriterRespV2) {

				defer close(stiwChan)
				stibs, processedDocs, err := stiw.ProcessSourceBlocks(blkList)

				if err != nil && err != io.EOF {
					stiwChan <- &SearchTermIdxWriterRespV2{stibs, processedDocs, err}
				} else {
					stiwChan <- &SearchTermIdxWriterRespV2{stibs, processedDocs, nil}
				}
			}(stiw, blkList, stiwChan)
		}

		// Wait for each parallel writing to finish
		respMap := make(map[*SearchTermIdxWriterV2]*SearchTermIdxWriterRespV2)
		for stiw, stiwChan := range stiwToChan {
			respMap[stiw] = <-stiwChan
		}

		stiWriteChan <- &StiWriteRespV2{respMap, nil}
	}
}

type StiCloseRespV2 struct {
	stiwErrorMap map[*SearchTermIdxWriterV2]error
	Error        error
}

func (batch *SearchTermBatchV2) stiCloseThread(writeChan <-chan *StiWriteRespV2, closeChan chan<- *StiCloseRespV2) {
	stiwErrorMap := make(map[*SearchTermIdxWriterV2]error)
	batch.processedDocs = make(processedTermIDTermDocs) // termID -> (term -> (docID -> docVer))
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
			batch.updateProcessedDocs(stiw.TermIDInfo.TermID, writeResp.ProcessedTermDocs)
		}
	} // Processing write channel

	type closeStiwResp struct {
		processedTermDocs processedTermDocs // term -> (docID -> docVer)
		err               error
	}

	// Close all STIW in parallel
	stiwToCloseChan := make(map[*SearchTermIdxWriterV2](chan *closeStiwResp))
	for _, termIDInfo := range batch.termIDInfoMap {
		stiwCloseChan := make(chan *closeStiwResp)
		stiwToCloseChan[termIDInfo.writer] = stiwCloseChan
		go func(stiw *SearchTermIdxWriterV2, stiwCloseChan chan *closeStiwResp) {
			defer close(stiwCloseChan)
			_, docs, err := stiw.Close()
			stiwCloseChan <- &closeStiwResp{docs, err}
		}(termIDInfo.writer, stiwCloseChan)
	}

	for stiw, stiwCloseChan := range stiwToCloseChan {
		// If there is already an error for this stiw, do not overwrite
		err := closeResp.Error
		if stiwErrorMap[stiw] == nil {
			stiwErrorMap[stiw] = err
		}

		closeResp := <-stiwCloseChan
		batch.updateProcessedDocs(stiw.TermIDInfo.TermID, closeResp.processedTermDocs)
	}

	closeChan <- &StiCloseRespV2{stiwErrorMap, nil}
}

// newProcessedDocs: term -> (docID -> docVer))
func (batch *SearchTermBatchV2) updateProcessedDocs(termID string, newProcessedTermDocs processedTermDocs) {
	// update processed docs
	if len(newProcessedTermDocs) > 0 {
		if _, ok := batch.processedDocs[termID]; !ok {
			batch.processedDocs[termID] = make(processedTermDocs)
		}

		for term, docIdVer := range newProcessedTermDocs {
			if _, ok := batch.processedDocs[termID][term]; !ok {
				batch.processedDocs[termID][term] = make(processedDocs)
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
//               Search Term Index Writer V2
//
//////////////////////////////////////////////////////////////////

type SearchTermIdxWriterRespV2 struct {
	Blocks            []*SearchTermIdxBlkV2
	ProcessedTermDocs processedTermDocs // term -> (docID -> docVer)
	Error             error
}

type SearchTermIdxWriterV2 struct {
	TermIDInfo       *SearchTermIDInfo
	OrigTerms        []string          // original terms
	origToAnalzdTerm map[string]string // Original Term -> Analyzed Term
	newSti           *SearchTermIdxV2
	oldSti           common.SearchTermIdx
	newStiBlk        *SearchTermIdxBlkV2
	oldStiBlk        *SearchTermIdxBlkV2
	updateID         string
	highDocVerMap    map[string]uint64          // DocID -> DocVer
	termDelDocMap    map[string]map[string]bool // Analyzed Term -> (DocID -> bool)
}

// sourceBlk -> (analzdTerm -> []offset)
type sourceTermOffsetState map[*SearchTermIdxSourceBlockV2]map[string][]uint64

func CreateSearchTermIdxWriterV2(sdc client.StrongDocClient,
	searchTermIDInfo *SearchTermIDInfo) (stiw *SearchTermIdxWriterV2, err error) {

	stiw = &SearchTermIdxWriterV2{
		TermIDInfo:       searchTermIDInfo,
		OrigTerms:        nil,
		origToAnalzdTerm: make(map[string]string), // Original Term -> Analyzed Term
		newSti:           nil,
		oldSti:           nil,
		newStiBlk:        nil,
		oldStiBlk:        nil,
		updateID:         "",
		highDocVerMap:    make(map[string]uint64),          // DocID -> DocVer
		termDelDocMap:    make(map[string]map[string]bool), // Analyzed Term -> (DocID -> bool)
	}

	for _, termInfo := range stiw.TermIDInfo.TermInfoList {
		stiw.OrigTerms = append(stiw.OrigTerms, termInfo.OrigTerm)
		stiw.origToAnalzdTerm[termInfo.OrigTerm] = termInfo.AnalzdTerm

		delDocMap := make(map[string]bool) // DocID -> bool
		stiw.termDelDocMap[termInfo.AnalzdTerm] = delDocMap
		if termInfo.DelSources != nil {
			for _, delSource := range termInfo.DelSources {
				delDocMap[delSource.GetDocID()] = true
			}
		}
	}

	// updateID of old STI
	oldUpdateID, _ := GetLatestUpdateIDV2(sdc, stiw.TermIDInfo.Owner, stiw.TermIDInfo.TermID)

	err = stiw.updateHighDocVersion(sdc, oldUpdateID)
	if err != nil {
		return nil, err
	}

	// Open previous STI if there is one
	if len(oldUpdateID) > 0 {
		stiw.oldSti, err = OpenSearchTermIdxV2(sdc, stiw.TermIDInfo.Owner, stiw.TermIDInfo.TermID,
			stiw.TermIDInfo.TermKey, stiw.TermIDInfo.IndexKey, oldUpdateID)
		if err != nil {
			stiw.oldSti = nil
		}
	}

	// Create new STI
	stiw.newSti, err = CreateSearchTermIdxV2(sdc, stiw.TermIDInfo.Owner, stiw.TermIDInfo.TermID,
		stiw.TermIDInfo.TermKey, stiw.TermIDInfo.IndexKey, nil, nil, stiw.TermIDInfo.TokenizerType)
	if err != nil {
		return nil, err
	}

	stiw.updateID = stiw.newSti.updateID
	return stiw, nil
}

// Get the highest document versions
func (stiw *SearchTermIdxWriterV2) updateHighDocVersion(sdc client.StrongDocClient, oldUpdateID string) error {
	for _, termInfo := range stiw.TermIDInfo.TermInfoList {
		// Find the highest version of each DocID. This is so we can remove
		// any DocID with lower version
		for _, addSource := range termInfo.AddSources {
			curVer, exist := stiw.highDocVerMap[addSource.GetDocID()]
			if !exist || curVer < addSource.GetDocVer() {
				stiw.highDocVerMap[addSource.GetDocID()] = addSource.GetDocVer()
			}
		}
		for _, delSource := range termInfo.DelSources {
			curVer, exist := stiw.highDocVerMap[delSource.GetDocID()]
			if !exist || curVer < delSource.GetDocVer() {
				stiw.highDocVerMap[delSource.GetDocID()] = delSource.GetDocVer()
			}
		}
	}

	if len(oldUpdateID) <= 0 {
		return nil
	}

	oldSsdi, err := OpenSearchSortDocIdxV2(sdc, stiw.TermIDInfo.Owner, stiw.TermIDInfo.TermID,
		stiw.TermIDInfo.TermKey, stiw.TermIDInfo.IndexKey, oldUpdateID)
	if err == nil { // read from old sortedDocIndex if exists
		defer oldSsdi.Close()
		err = nil
		for err == nil {
			var oldBlk *SearchSortDocIdxBlkV2 = nil
			oldBlk, err = oldSsdi.ReadNextBlock()
			if err != nil && err != io.EOF {
				return err
			}
			if oldBlk == nil {
				continue
			}
			for _, oldDocIDVer := range oldBlk.termToDocIDVer {
				for oldDocID, oldDocVer := range oldDocIDVer {
					// Only put older document versions in the highDocVerMap if it's actually higher
					// than the current document version
					curVer, exist := stiw.highDocVerMap[oldDocID]
					if exist && curVer < oldDocVer {
						stiw.highDocVerMap[oldDocID] = oldDocVer
					}
				}
			}
		}
	} else { // read from old searchTermIndex if exists
		oldSti, err := OpenSearchTermIdxV2(sdc, stiw.TermIDInfo.Owner, stiw.TermIDInfo.TermID,
			stiw.TermIDInfo.TermKey, stiw.TermIDInfo.IndexKey, oldUpdateID)
		if err != nil {
			stiw.oldSti = nil
		}
		defer oldSti.Close()

		err = nil
		for err == nil {
			var oldBlk *SearchTermIdxBlkV2 = nil
			oldBlk, err = oldSti.ReadNextBlock()
			if err != nil && err != io.EOF {
				return err
			}
			if oldBlk == nil {
				continue
			}
			for _, oldDocIDVer := range oldBlk.TermDocVerOffset {
				for oldDocID, oldDocVerOff := range oldDocIDVer {
					// Only put older document versions in the highDocVerMap if it's actually higher
					// than the current document version
					curVer, exist := stiw.highDocVerMap[oldDocID]
					if exist && curVer < oldDocVerOff.Version {
						stiw.highDocVerMap[oldDocID] = oldDocVerOff.Version
					}
				}
			}
		}
	}

	return nil
}

// 1. Remove all sources block with lower version than highest document version
// 2. Remove all sources block in source block delete list
// 3. Remove all sources block in global delete list
func (stiw *SearchTermIdxWriterV2) shouldRemoveDoc(docID string, docVer uint64) bool {
	highVer, exist := stiw.highDocVerMap[docID]
	if (exist && highVer > docVer) ||
		(stiw.TermIDInfo.globalDelDocs != nil && stiw.TermIDInfo.globalDelDocs.delDocMap[docID]) {
		return true
	}

	return false
}

// TODO: optimize by parallelism
// process one bucket writer
func (stiw *SearchTermIdxWriterV2) ProcessSourceBlocks(sourceBlocks []*SearchTermIdxSourceBlockV2) (
	returnBlocks []*SearchTermIdxBlkV2, // returned blocks
	procTermDocs processedTermDocs, // term -> (docID -> docVer)
	err error) {

	procTermDocs = make(processedTermDocs)

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

	srcTermOffsetState := stiw.getSourceTermOffsetState(sourceBlocks)
	// The state for the following is kept in srcTermOffsetState
	// 1. Process a limited amount of offsets in source blocks
	// 2. Process a limited amount of old STI blocks
	// 3. Repeat until all the source blocks are empty
	// 4. If the old STI blocks are not empty, it will wait for
	//    next set of source blocks, or for STIW close to write
	//    the rest
	for sourceNotEmpty := true; sourceNotEmpty; {
		// TODO: optimization parallel process terms
		var blks []*SearchTermIdxBlkV2 = nil
		var processed processedTermDocs // term -> (docID -> docVer)
		sourceNotEmpty, processed, blks, err = stiw.processSourceBlocks(sourceBlocks,
			srcTermOffsetState, copyLen)
		if err != nil {
			return nil, nil, err
		}
		for term, docIDVer := range processed {
			for id, ver := range docIDVer {
				if _, ok := procTermDocs[term]; !ok {
					procTermDocs[term] = make(map[string]uint64)
				}
				procTermDocs[term][id] = ver

			}
		}

		returnBlocks = append(returnBlocks, blks...)

		blks, processed, err = stiw.processOldStiBlock(copyLen, false)
		if err != nil && err != io.EOF {
			return nil, nil, err
		}
		for id, ver := range processed {
			procTermDocs[id] = ver
		}
		returnBlocks = append(returnBlocks, blks...)
	}

	return returnBlocks, procTermDocs, err
}

func (stiw *SearchTermIdxWriterV2) getSourceTermOffsetState(sourceBlocks []*SearchTermIdxSourceBlockV2) sourceTermOffsetState {
	// sourceBlk -> (analzdTerm -> []offset)
	srcTermOffsetState := make(sourceTermOffsetState)

	// filter out old source document blocks, and merge + sort offsets for analyzed terms
	for _, srcBlk := range sourceBlocks {
		if stiw.shouldRemoveDoc(srcBlk.DocID, srcBlk.DocVer) {
			continue
		}

		// tracks which term location need sorting after
		needSort := make(map[string]bool) // AnalyzedTerm -> bool

		for _, origTerm := range stiw.OrigTerms {
			srcOffsets := srcBlk.TermOffset[origTerm]
			if len(srcOffsets) <= 0 {
				continue
			}

			analzdTerm := stiw.origToAnalzdTerm[origTerm]
			if _, ok := srcTermOffsetState[srcBlk]; !ok {
				srcTermOffsetState[srcBlk] = make(map[string][]uint64) // analzdTerm -> []offset
			}

			curOffsets := srcTermOffsetState[srcBlk][analzdTerm]

			if len(curOffsets) == 0 {
				srcTermOffsetState[srcBlk][analzdTerm] = srcOffsets
			} else {
				// The resulting offsets will be out of order. Need to sort after
				if curOffsets[len(curOffsets)-1] > srcOffsets[0] {
					needSort[analzdTerm] = true
				}
				srcTermOffsetState[srcBlk][analzdTerm] = append(srcTermOffsetState[srcBlk][analzdTerm],
					srcOffsets...)
			}
		} // for _, origTerm := range stiw.OrigTerms

		for sortAnalzdTerm := range needSort {
			sortOffsets := srcTermOffsetState[srcBlk][sortAnalzdTerm]
			sort.Slice(sortOffsets, func(i, j int) bool { return sortOffsets[i] < sortOffsets[j] })
		}
	} // for _, srcBlk := range sourceBlocks

	return srcTermOffsetState
}

func (stiw *SearchTermIdxWriterV2) processSourceBlocks(sourceBlocks []*SearchTermIdxSourceBlockV2,
	srcTermOffsetState sourceTermOffsetState, copyLen int) (
	sourceNotEmpty bool,
	procTermDocs processedTermDocs,
	writtenBlocks []*SearchTermIdxBlkV2,
	err error) {

	sourceNotEmpty = false
	writtenBlocks = make([]*SearchTermIdxBlkV2, 0, 10)
	procTermDocs = make(processedTermDocs)
	err = nil

	// Go through each source document block, and process a limited amount of offsets
	// Remove the processed offsets from srcTermOffsetState
	for srcBlk, termSrcOffsets := range srcTermOffsetState {
		if len(termSrcOffsets) == 0 {
			delete(srcTermOffsetState, srcBlk)
			continue
		}

		for analzdTerm, srcOffsets := range termSrcOffsets {
			if len(srcOffsets) == 0 {
				delete(termSrcOffsets, analzdTerm)
				continue
			}
			count := utils.Min(len(srcOffsets), copyLen)
			offsets := srcOffsets[:count]

			err = stiw.newStiBlk.AddDocOffsets(analzdTerm, srcBlk.DocID, srcBlk.DocVer, offsets)
			if err != nil {
				return
			}

			if _, ok := procTermDocs[analzdTerm]; !ok {
				procTermDocs[analzdTerm] = make(map[string]uint64)
			}
			procTermDocs[analzdTerm][srcBlk.DocID] = srcBlk.DocVer

			srcTermOffsetState[srcBlk][analzdTerm] = srcOffsets[count:]
			if len(srcTermOffsetState[srcBlk][analzdTerm]) > 0 {
				sourceNotEmpty = true
			}

			// Current block is full. Send it
			if stiw.newStiBlk.IsFull() {
				err = stiw.newSti.WriteNextBlock(stiw.newStiBlk)
				if err != nil {
					return
				}

				writtenBlocks = append(writtenBlocks, stiw.newStiBlk)
				stiw.newStiBlk = CreateSearchTermIdxBlkV2(stiw.newSti.GetMaxBlockDataSize())
			}
		} // for analzdTerm, srcOffsets := range termSrcOffsets
	} // for srcBlk, termSrcOffsets := range srcTermOffsetState

	return
}

// continueWrite = true means the code will continue will create a new output STI block
// when the old one fills up. During normal writing, continueWrite = false. This is
// because we want the new document information to appear first in the new output
// STI block. So continueWrite = true is only used when there are no more new document
// information left, and we're closing the STIW. At that time we're just trying
// to read all remaining information from the old STI and transfer it to the new
// STI
func (stiw *SearchTermIdxWriterV2) processOldStiBlock(copyLen int, continueWrite bool) (
	writtenBlocks []*SearchTermIdxBlkV2,
	procTermDocs processedTermDocs, // analyzed term -> (docID -> docVersion)
	err error) {

	err = nil
	writtenBlocks = make([]*SearchTermIdxBlkV2, 0, 10)
	procTermDocs = make(processedTermDocs)

	if stiw.oldStiBlk == nil { // No more old STI blocks to process
		return
	}

	stopWriting := false

	// Go through the old STI block, and process a limited amount of offsets
	// Remove the processed offsets from old STI block
	for oldAnalzdTerm, oldDocVerOffset := range stiw.oldStiBlk.TermDocVerOffset {
		if len(oldDocVerOffset) == 0 {
			delete(stiw.oldStiBlk.TermDocVerOffset, oldAnalzdTerm)
			continue
		}

		for oldDocID, oldVerOffset := range oldDocVerOffset {
			if oldVerOffset == nil || len(oldVerOffset.Offsets) == 0 {
				delete(oldDocVerOffset, oldDocID)
				continue
			}

			if stiw.shouldRemoveDoc(oldDocID, oldVerOffset.Version) ||
				stiw.termDelDocMap[oldAnalzdTerm][oldDocID] {
				delete(oldDocVerOffset, oldDocID)
				continue
			}

			// Write oldSTI to newSTI
			count := utils.Min(len(oldVerOffset.Offsets), copyLen)
			offsets := oldVerOffset.Offsets[:count]

			err = stiw.newStiBlk.AddDocOffsets(oldAnalzdTerm, oldDocID, oldVerOffset.Version, offsets)
			if err != nil {
				return
			}

			// update processedDocs
			if _, ok := procTermDocs[oldAnalzdTerm]; !ok {
				procTermDocs[oldAnalzdTerm] = make(map[string]uint64)
			}
			procTermDocs[oldAnalzdTerm][oldDocID] = oldVerOffset.Version

			// Remove the successfully processed offsets from the old STI block (in-place change)
			oldVerOffset.Offsets = oldVerOffset.Offsets[count:]

			// Current block is full. Send it
			if stiw.newStiBlk.IsFull() {
				err = stiw.newSti.WriteNextBlock(stiw.newStiBlk)
				if err != nil {
					return
				}
				writtenBlocks = append(writtenBlocks, stiw.newStiBlk)
				stiw.newStiBlk = CreateSearchTermIdxBlkV2(stiw.newSti.GetMaxBlockDataSize())

				if !continueWrite {
					stopWriting = true
					break
				}
			}
		} // for oldDocID, oldVerOffset := range oldDocVerOffset

		if stopWriting {
			break
		}
	} // for oldAnalzdTerm, oldDocVerOffset := range stiw.oldStiBlk.TermDocVerOffset

	if len(stiw.oldStiBlk.TermDocVerOffset) == 0 { // all docIDs are processed, get next old block
		err = stiw.getOldSearchTermIndexBlock()
		if err != nil && err != io.EOF {
			return
		}
	}

	return
}

// read one block from old search term index
// set stiw.OldStiBlock to the original block read from STI
func (stiw *SearchTermIdxWriterV2) getOldSearchTermIndexBlock() (err error) {
	err = nil
	if stiw.oldSti != nil {
		stiw.oldStiBlk = nil
		oldSti := stiw.oldSti.(*SearchTermIdxV2)
		stiw.oldStiBlk, err = oldSti.ReadNextBlock()
	}

	return err
}

func (stiw *SearchTermIdxWriterV2) Close() (
	returnBlocks []*SearchTermIdxBlkV2,
	procTermDocs processedTermDocs, // analzdTerm -> (docID -> docVer)
	finalErr error) {

	returnBlocks = make([]*SearchTermIdxBlkV2, 0, 10)
	var err error = nil
	finalErr = nil

	if stiw.oldStiBlk != nil {
		for err != nil {
			var blks []*SearchTermIdxBlkV2 = nil
			blks, procTermDocs, err = stiw.processOldStiBlock(copyLen, true)
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

	return returnBlocks, procTermDocs, finalErr
}

func (stiw *SearchTermIdxWriterV2) GetUpdateID() string {
	return stiw.updateID
}

func (batch *SearchTermBatchV2) IsEmpty() bool {
	return (len(batch.termIDInfoMap) == 0)
}
