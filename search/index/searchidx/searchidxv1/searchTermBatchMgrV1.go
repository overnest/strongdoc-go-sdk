package searchidxv1

import (
	"io"
	"sort"
	"strings"

	"github.com/emirpasic/gods/trees/binaryheap"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

var copyLen int = 10

//////////////////////////////////////////////////////////////////
//
//                   Search HashedTerm Batch Manager V1
//
//////////////////////////////////////////////////////////////////

type SearchTermBatchMgrV1 struct {
	Owner            common.SearchIdxOwner
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

func CreateSearchTermBatchMgrV1(owner common.SearchIdxOwner, sources []SearchTermIdxSourceV1,
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

func (mgr *SearchTermBatchMgrV1) GetNextTermBatch(sdc client.StrongDocClient, batchSize int) (*SearchTermBatchV1, error) {
	sources := make([]*SearchTermBatchSources, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		s, _ := mgr.termHeap.Pop()
		if s == nil {
			break
		}

		stbs := s.(*SearchTermBatchSources)
		sources = append(sources, stbs)
	}

	return CreateSearchTermBatchV1(sdc, mgr.Owner, sources)
}

func (mgr *SearchTermBatchMgrV1) ProcessNextTermBatch(sdc client.StrongDocClient, batchSize int) (map[string]error, error) {
	batchResult := make(map[string]error)

	termBatch, err := mgr.GetNextTermBatch(sdc, batchSize)
	if err != nil {
		return batchResult, err
	}
	defer termBatch.Close()

	if termBatch.IsEmpty() {
		return batchResult, io.EOF
	}

	return termBatch.ProcessTermBatch(sdc, nil)
}

func (mgr *SearchTermBatchMgrV1) ProcessAllTermBatches(sdc client.StrongDocClient, batchSize int) (map[string]error, error) {
	finalResult := make(map[string]error)

	var err error = nil
	for err == nil {
		var batchResult map[string]error

		batchResult, err = mgr.ProcessNextTermBatch(sdc, batchSize)
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
//                    Search HashedTerm Batch V1
//
//////////////////////////////////////////////////////////////////

type SearchTermBatchV1 struct {
	Owner           common.SearchIdxOwner
	SourceList      []SearchTermIdxSourceV1
	TermToWriter    map[string]*SearchTermIdxWriterV1 // term -> SearchTermIdxWriterV1
	sourceToWriters map[SearchTermIdxSourceV1][]*SearchTermIdxWriterV1
	termList        []string
}

func CreateSearchTermBatchV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, sources []*SearchTermBatchSources) (*SearchTermBatchV1, error) {
	batch := &SearchTermBatchV1{
		Owner:           owner,
		SourceList:      nil,
		TermToWriter:    make(map[string]*SearchTermIdxWriterV1),
		sourceToWriters: make(map[SearchTermIdxSourceV1][]*SearchTermIdxWriterV1),
		termList:        nil,
	}

	type stiwResult struct {
		writer *SearchTermIdxWriterV1
		error  error
	}

	stiwChans := make([]chan stiwResult, len(sources))
	for i, source := range sources {
		stiwChan := make(chan stiwResult)
		stiwChans[i] = stiwChan
		go func(sdc client.StrongDocClient,
			owner common.SearchIdxOwner, source *SearchTermBatchSources, batch *SearchTermBatchV1,
			stiwChan chan<- stiwResult) {
			defer close(stiwChan)
			stiw, err := CreateSearchTermIdxWriterV1(
				sdc, owner, source.Term, source.AddSources, source.DelSources,
				source.TermKey, source.IndexKey, source.delDocs)
			stiwChan <- stiwResult{
				stiw,
				err,
			}
		}(sdc, owner, source, batch, stiwChan)
	}

	for _, stiwChan := range stiwChans {
		res := <-stiwChan
		if res.error != nil {
			return nil, res.error
		}
		batch.TermToWriter[res.writer.Term] = res.writer
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

func (batch *SearchTermBatchV1) ProcessTermBatch(sdc client.StrongDocClient, event *utils.TimeEvent) (map[string]error, error) {
	respMap := make(map[string]error)

	e1 := utils.AddSubEvent(event, "processStiAll")
	processDocs, stiResp, err := batch.processStiAll(e1)
	if err != nil {
		return respMap, err
	}
	utils.EndEvent(e1)

	stiSuccess := make([]*SearchTermIdxWriterV1, 0, len(stiResp))
	for stiw, err := range stiResp {
		respMap[stiw.Term] = utils.FirstError(respMap[stiw.Term], err)
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
		respMap[stiw.Term] = utils.FirstError(respMap[stiw.Term], err)
	}
	utils.EndEvent(e2)

	return respMap, nil
}

type SearchTermBlockRespV1 struct {
	Block *SearchTermIdxSourceBlockV1
	Error error
}

type SearchTermReadingRespV1 struct {
	stiwToBlks map[*SearchTermIdxWriterV1][]*SearchTermIdxSourceBlockV1
	Error      error
}

type SearchTermWritingRespV1 struct {
	stiwToResp map[*SearchTermIdxWriterV1]*SearchTermIdxWriterRespV1
	Error      error
}

type SearchTermCollectingRespV1 struct {
	respMap       map[*SearchTermIdxWriterV1]error
	processedDocs map[string](map[string]uint64)
	Error         error
}

func (batch *SearchTermBatchV1) processStiBatchParaReading(readToWriteChan chan<- *SearchTermReadingRespV1) {

	defer close(readToWriteChan)

	for {
		stiwToBlks := make(map[*SearchTermIdxWriterV1][]*SearchTermIdxSourceBlockV1)
		blkToChan := make(map[SearchTermIdxSourceV1](chan *SearchTermBlockRespV1))

		// Start parallel threads to read
		for _, source := range batch.SourceList {

			blkChan := make(chan *SearchTermBlockRespV1)
			blkToChan[source] = blkChan

			go func(source SearchTermIdxSourceV1, batch *SearchTermBatchV1, blkChan chan<- *SearchTermBlockRespV1) {
				defer close(blkChan)
				blk, err := source.GetNextSourceBlock(batch.termList)
				blkChan <- &SearchTermBlockRespV1{blk, err}
			}(source, batch, blkChan)
		}

		for source, blkChan := range blkToChan {
			blkResp := <-blkChan
			err := blkResp.Error
			blk := blkResp.Block

			if err != nil && err != io.EOF {
				readToWriteChan <- &SearchTermReadingRespV1{stiwToBlks, err}
				return
			}

			if blk != nil && len(blk.TermOffset) > 0 {
				// doc1 -> [term_A_writer, term_B_writer, term_C_writer]
				for _, stiw := range batch.sourceToWriters[source] {
					blkList := stiwToBlks[stiw]
					if blkList == nil {
						blkList = make([]*SearchTermIdxSourceBlockV1, 0, len(stiw.AddSources))
					}
					stiwToBlks[stiw] = append(blkList, blk)
				}
			}
		}

		// term_A_writer -> [doc1_block2, doc3_block2, doc5_block2] ... but not guarantee block contains term A

		// Finish reading all blocks
		if len(stiwToBlks) == 0 {
			readToWriteChan <- &SearchTermReadingRespV1{stiwToBlks, nil}
			return
		}

		// Finish reading one more batch, not finish all yet, just go on with the loop
		readToWriteChan <- &SearchTermReadingRespV1{stiwToBlks, nil}
	}

}

func (batch *SearchTermBatchV1) processStiBatchParaWriting(readToWriteChan <-chan *SearchTermReadingRespV1, writeToCollectChan chan<- *SearchTermWritingRespV1) {
	defer close(writeToCollectChan)

	for {
		readResp, more := <-readToWriteChan

		if more {

			// Parse the response from reading
			stiwToBlks := readResp.stiwToBlks
			err := readResp.Error

			// Check errors
			if err != nil {
				writeToCollectChan <- &SearchTermWritingRespV1{nil, err}
				return
			}

			// Process Source Block in Parallel
			stiwToChan := make(map[*SearchTermIdxWriterV1](chan *SearchTermIdxWriterRespV1))
			for stiw, blkList := range stiwToBlks {
				if len(blkList) > 0 {
					stiwChan := make(chan *SearchTermIdxWriterRespV1)
					stiwToChan[stiw] = stiwChan

					// This executes in a separate thread
					go func(stiw *SearchTermIdxWriterV1, blkList []*SearchTermIdxSourceBlockV1,
						stiwChan chan<- *SearchTermIdxWriterRespV1) {
						defer close(stiwChan)

						stibs, processedDocs, err := stiw.ProcessSourceBlocks(blkList)
						stiwChan <- &SearchTermIdxWriterRespV1{stibs, processedDocs, err}
					}(stiw, blkList, stiwChan)
				}
			}

			// Wait for each parallel writing to finish
			respMap := make(map[*SearchTermIdxWriterV1]*SearchTermIdxWriterRespV1)
			for stiw, stiwChan := range stiwToChan {
				respMap[stiw] = <-stiwChan
			}

			writeToCollectChan <- &SearchTermWritingRespV1{respMap, nil}

		} else {
			return
		}

	}

}

func (batch *SearchTermBatchV1) processStiBatchParaCollecting(writeToCollectChan <-chan *SearchTermWritingRespV1, collectToAllChan chan<- *SearchTermCollectingRespV1) {
	respMap := make(map[*SearchTermIdxWriterV1]error)
	processedDocs := make(map[string](map[string]uint64))
	defer close(collectToAllChan)

	for {
		writingResp, more := <-writeToCollectChan
		if more {
			stiBlocksResp := writingResp.stiwToResp
			err := writingResp.Error

			if err != nil && err != io.EOF {
				collectToAllChan <- &SearchTermCollectingRespV1{respMap, processedDocs, err}
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
							if processedDocs[stiw.Term] == nil {
								processedDocs[stiw.Term] = resp.ProcessedDocs
							} else {
								for id, ver := range resp.ProcessedDocs {
									processedDocs[stiw.Term][id] = ver
								}
							}
						}
					} else {
						respMap[stiw] = nil
					}
				}
			}
		} else {
			collectToAllChan <- &SearchTermCollectingRespV1{respMap, processedDocs, nil}
			return
		}

	}
}

func (batch *SearchTermBatchV1) processStiAll(event *utils.TimeEvent) (
	map[string](map[string]uint64),
	map[*SearchTermIdxWriterV1]error, error) {
	e1 := utils.AddSubEvent(event, "resetSources")
	for _, source := range batch.SourceList {
		source.Reset()
	}
	utils.EndEvent(e1)

	e2 := utils.AddSubEvent(event, "processStiBatches in 3 threads")

	// Prepare Channels for reading -> writing -> stiAll
	readToWriteChan := make(chan *SearchTermReadingRespV1)
	writeToCollectChan := make(chan *SearchTermWritingRespV1)
	collectToAllChan := make(chan *SearchTermCollectingRespV1)

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
		processedDocs map[string]uint64
		err           error
	}

	stiwToChan := make(map[*SearchTermIdxWriterV1](chan closeStiwResp))
	for _, stiw := range batch.TermToWriter {
		stiwChan := make(chan closeStiwResp)
		stiwToChan[stiw] = stiwChan
		go func(stiw *SearchTermIdxWriterV1, stiwChan chan closeStiwResp) {
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
			if processedDocs != nil && len(processedDocs) > 0 {
				if processedDocs[stiw.Term] == nil {
					processedDocs[stiw.Term] = resp.processedDocs
				} else {
					for id, ver := range resp.processedDocs {
						processedDocs[stiw.Term][id] = ver
					}
				}
			}
		}
	}
	utils.EndEvent(e3)
	return processedDocs, respMap, nil
}

func (batch *SearchTermBatchV1) processSsdiAll(sdc client.StrongDocClient, stiwList []*SearchTermIdxWriterV1, processedDocs map[string](map[string]uint64)) (map[*SearchTermIdxWriterV1]error, error) {
	ssdiToChan := make(map[*SearchTermIdxWriterV1](chan error))

	for _, stiw := range stiwList {
		ssdiChan := make(chan error)
		ssdiToChan[stiw] = ssdiChan

		// This executes in a separate thread
		go func(stiw *SearchTermIdxWriterV1, docs map[string]uint64, ssdiChan chan<- error) {
			defer close(ssdiChan)
			ssdi, err := CreateSearchSortDocIdxV1(sdc, batch.Owner, stiw.Term,
				stiw.GetUpdateID(), stiw.TermKey, stiw.IndexKey, docs)
			if err != nil {
				ssdiChan <- err
				return
			}
			defer ssdi.Close()

			for err == nil {
				var blk *SearchSortDocIdxBlkV1
				blk, err = ssdi.WriteNextBlock()

				_ = blk
				// fmt.Println(stiw.HashedTerm, blk.DocIDVers)

				if err != nil && err != io.EOF {
					ssdiChan <- err
					return
				}
			}

			ssdi.Close()
			ssdiChan <- nil
		}(stiw, processedDocs[stiw.Term], ssdiChan)
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
//                 Search HashedTerm Batch Writer V1
//
//////////////////////////////////////////////////////////////////

type SearchTermIdxWriterRespV1 struct {
	Blocks        []*SearchTermIdxBlkV1
	ProcessedDocs map[string]uint64
	Error         error
}

type SearchTermIdxWriterV1 struct {
	Term            string
	Owner           common.SearchIdxOwner
	AddSources      []SearchTermIdxSourceV1
	DelSources      []SearchTermIdxSourceV1
	TermKey         *sscrypto.StrongSaltKey
	IndexKey        *sscrypto.StrongSaltKey
	updateID        string
	delDocMap       map[string]bool   // DocID -> boolean
	highDocVerMap   map[string]uint64 // DocID -> DocVer
	newSti          *SearchTermIdxV1
	oldSti          common.SearchTermIdx
	newStiBlk       *SearchTermIdxBlkV1
	oldStiBlk       *SearchTermIdxBlkV1
	oldStiBlkDocIDs []string
}

func CreateSearchTermIdxWriterV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string,
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
	updateID, _ := GetLatestUpdateIDV1(sdc, owner, term, termKey)
	if updateID != "" {
		stiw.oldSti, err = OpenSearchTermIdxV1(sdc, owner, term, termKey, indexKey, updateID)
		if err != nil {
			stiw.oldSti = nil
		}
		err = stiw.updateHighDocVersion(sdc, owner, term, termKey, indexKey, updateID)
		if err != nil {
			return nil, err
		}
	}

	stiw.newSti, err = CreateSearchTermIdxV1(sdc, owner, term, termKey, indexKey, nil, nil)
	if err != nil {
		return nil, err
	}

	stiw.updateID = stiw.newSti.updateID
	return stiw, nil
}

func (stiw *SearchTermIdxWriterV1) updateHighDocVersion(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string,
	termKey, indexKey *sscrypto.StrongSaltKey, updateID string) error {
	ssdi, err := OpenSearchSortDocIdxV1(sdc, owner, term, termKey, indexKey, updateID)
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

func (stiw *SearchTermIdxWriterV1) ProcessSourceBlocks(sourceBlocks []*SearchTermIdxSourceBlockV1) (
	[]*SearchTermIdxBlkV1, map[string]uint64, error) {

	var err error
	returnBlocks := make([]*SearchTermIdxBlkV1, 0, len(sourceBlocks)*2)
	cloneSourceBlocks := make([]*SearchTermIdxSourceBlockV1, 0, len(sourceBlocks))
	processedDocs := make(map[string]uint64)
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
			return nil, nil, err
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
		var processed map[string]uint64
		sourceNotEmpty, processed, blks, err = stiw.processSourceBlocks(cloneSourceBlocks, srcOffsetMap, copyLen)
		if err != nil {
			return nil, nil, err
		}
		for id, ver := range processed {
			processedDocs[id] = ver
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
	} // for sourceNotEmpty := true; sourceNotEmpty;

	return returnBlocks, processedDocs, nil
}

func (stiw *SearchTermIdxWriterV1) processSourceBlocks(
	sourceBlocks []*SearchTermIdxSourceBlockV1,
	srcOffsetMap map[*SearchTermIdxSourceBlockV1][]uint64,
	copyLen int) (bool, map[string]uint64, []*SearchTermIdxBlkV1, error) {

	sourceNotEmpty := false
	var err error = nil
	returnBlocks := make([]*SearchTermIdxBlkV1, 0, 10)
	processedDocs := make(map[string]uint64)

	// Process the source blocks
	for _, srcBlk := range sourceBlocks {
		srcOffsets := srcOffsetMap[srcBlk]
		if len(srcOffsets) == 0 {
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
				return sourceNotEmpty, processedDocs, returnBlocks, err
			}

			returnBlocks = append(returnBlocks, stiw.newStiBlk)

			stiw.newStiBlk = CreateSearchTermIdxBlkV1(stiw.newSti.GetMaxBlockDataSize())
			err = stiw.newStiBlk.AddDocOffsets(srcBlk.DocID, srcBlk.DocVer, offsets)
			if err != nil {
				return sourceNotEmpty, processedDocs, returnBlocks, err
			}
		}
		processedDocs[srcBlk.DocID] = srcBlk.DocVer
		srcOffsetMap[srcBlk] = srcOffsets[count:]
	} // for _, srcBlk := range sourceBlocks

	return sourceNotEmpty, processedDocs, returnBlocks, nil
}

func (stiw *SearchTermIdxWriterV1) processOldStiBlock(copyLen int, writeFull bool) ([]*SearchTermIdxBlkV1, map[string]uint64, error) {
	var err error = nil
	returnBlocks := make([]*SearchTermIdxBlkV1, 0, 10)
	processedDocs := make(map[string]uint64)
	if stiw.oldStiBlk == nil {
		// No more old STI blocks to process
		return returnBlocks, processedDocs, io.EOF
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
					return returnBlocks, processedDocs, err
				}

				returnBlocks = append(returnBlocks, stiw.newStiBlk)

				stiw.newStiBlk = CreateSearchTermIdxBlkV1(stiw.newSti.GetMaxBlockDataSize())
				err = stiw.newStiBlk.AddDocOffsets(docID, verOffset.Version, offsets)
				if err != nil {
					return returnBlocks, processedDocs, err
				}
			} else {
				// The new block is full. Don't bother processing anymore old STI block
				break
			}
		}
		processedDocs[docID] = verOffset.Version

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
			return returnBlocks, processedDocs, err
		}
	}

	return returnBlocks, processedDocs, err
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

func (stiw *SearchTermIdxWriterV1) Close() ([]*SearchTermIdxBlkV1, map[string]uint64, error) {
	returnBlocks := make([]*SearchTermIdxBlkV1, 0, 10)
	var blks []*SearchTermIdxBlkV1 = nil
	var err, finalErr error = nil, nil
	var processedDocs map[string]uint64

	if stiw.oldStiBlk != nil {
		// all old sti blocks are processed!!!
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
