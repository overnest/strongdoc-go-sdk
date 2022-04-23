package searchidx

import (
	"io"
	"os"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/searchidxv1"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/searchidxv2"
	"github.com/overnest/strongdoc-go-sdk/search/tokenizer"
	"github.com/overnest/strongdoc-go-sdk/utils"

	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
)

const (
	STI_EOF = uint64(0)
)

//////////////////////////////////////////////////////////////////
//
//                     Get Term Tokenizer
//
//////////////////////////////////////////////////////////////////
func GetSearchTermAnalyzer() (tokenizer.Analyzer, error) {
	return tokenizer.OpenAnalyzer(tokenizer.TKZER_BLEVE)
}

//////////////////////////////////////////////////////////////////
//
//                     Search Term Index
//
//////////////////////////////////////////////////////////////////

type StiDocOffsets interface {
	GetDocIDs() []string
	GetDocVer(docID string) uint64
	GetOffsets(docID string) []uint64
	GetDocIdOffsets() map[string][]uint64 // DocID -> Offsets
}

type StiData interface {
	IsTermEOF(term string) bool
	GetTermDocOffsets(term string) StiDocOffsets
	GetTermsDocOffsets(terms []string) map[string]StiDocOffsets // term -> StiDocOffsets
	GetAllDocOffsets() map[string]StiDocOffsets                 // term -> StiDocOffsets
}

type StiReader interface {
	ReadNextData() (StiData, error)
	GetTerms() []string
	Reset() error
	Close() error
}

type stiDocOffsets struct {
	docID   string
	docVer  uint64
	offsets []uint64
}

type stiTermDoc struct {
	docIDs     []string
	docOffsets map[string]*stiDocOffsets // docID -> stiDocOffsets
}

type stiData struct {
	termDoc map[string]StiDocOffsets // term -> StiDocOffsets
	termEof map[string]bool          // term -> eof(bool)
}

type stiReader struct {
	reader        io.ReadCloser
	size          uint64
	version       uint32
	termID        string
	analzdTerms   []string            // analyzed terms
	origTerms     []string            // original terms
	origToAnalzd  map[string]string   // original term -> analyzed term
	analzdToOrigs map[string][]string // analyzed term -> []original term
	analyzer      tokenizer.Analyzer
	stiV1         *searchidxv1.SearchTermIdxV1
	stiV2         *searchidxv2.SearchTermIdxV2
}

type stiBulkReader struct {
	analzdTerms []string
	stiReaders  []*stiReader
	analyzer    tokenizer.Analyzer
}

func OpenSearchTermIndex(sdc client.StrongDocClient, owner common.SearchIdxOwner, origTerms []string,
	termKey, indexKey *sscrypto.StrongSaltKey, stiVersion uint32) (StiReader, error) {

	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The term key type %v is not a MAC key", termKey.Type.Name)
	}

	if indexKey.Type != sscrypto.Type_XChaCha20 {
		return nil, errors.Errorf("Index key type %v is not supported. The only supported key type is %v",
			indexKey.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	analyzer, err := GetSearchTermAnalyzer()
	if err != nil {
		return nil, err
	}

	analzdTerms, _, analzdToOrigs, err := common.AnalyzeTerms(origTerms, analyzer)
	if err != nil {
		return nil, err
	}

	// termID -> list of terms
	termIDMap, err := common.GetTermIDs(analzdTerms, termKey, common.STI_TERM_BUCKET_COUNT, stiVersion)
	if err != nil {
		return nil, err
	}

	bulkReader := &stiBulkReader{
		analzdTerms: analzdTerms,
		stiReaders:  make([]*stiReader, 0, len(termIDMap)),
		analyzer:    analyzer,
	}

	for termID, analyzedTerms := range termIDMap {
		var readerOrigTerms []string = nil
		for _, analyzedTerm := range analyzedTerms {
			readerOrigTerms = append(readerOrigTerms, analzdToOrigs[analyzedTerm]...)
		}
		err := bulkReader.initStiReader(sdc, owner, termID, readerOrigTerms,
			termKey, indexKey, stiVersion)
		if err != nil && err != os.ErrNotExist {
			return nil, err
		}
	} // for termID, analyzedTerms := range termIDMap

	return bulkReader, nil
}

func (bReader *stiBulkReader) initStiReader(sdc client.StrongDocClient, owner common.SearchIdxOwner,
	termID string, origTerms []string, termKey, indexKey *sscrypto.StrongSaltKey, stiVersion uint32) error {

	updateID, err := common.GetLatestUpdateID(sdc, owner, termID)
	if err != nil {
		return err
	}

	reader, size, err := common.OpenSearchTermIndexReader(sdc, owner, termID, updateID)
	if err != nil {
		return err
	}

	stiRdr := &stiReader{
		reader:        reader,
		size:          size,
		version:       stiVersion,
		termID:        termID,
		analzdTerms:   nil,
		origTerms:     origTerms,
		origToAnalzd:  make(map[string]string),
		analzdToOrigs: make(map[string][]string),
	}

	plainHdr, plainHdrSize, err := ssheaders.DeserializePlainHdrStream(reader)
	if err != nil {
		return err
	}

	plainHdrBodyData, err := plainHdr.GetBody()
	if err != nil {
		return err
	}

	version, err := common.DeserializeStiVersion(plainHdrBodyData)
	if err != nil {
		return err
	}

	switch version.GetStiVersion() {
	case common.STI_V1:
		stiRdr.version = common.STI_V1

		// Parse plaintext header body
		plainHdrBody := &searchidxv1.StiPlainHdrBodyV1{}
		plainHdrBody, err := plainHdrBody.Deserialize(plainHdrBodyData)
		if err != nil {
			return errors.New(err)
		}

		stiRdrAnalyzer, err := GetSearchTermAnalyzer()
		if err != nil {
			return err
		}

		analyzedTerms := stiRdrAnalyzer.Analyze(origTerms[0])
		stiv1, err := searchidxv1.OpenSearchTermIdxPrivV1(owner, analyzedTerms[0], termID, updateID, termKey,
			indexKey, reader, plainHdr, 0, uint64(plainHdrSize), size)
		if err != nil {
			return err
		}

		stiRdr.analyzer = stiRdrAnalyzer
		stiRdr.stiV1 = stiv1
	case common.STI_V2:
		stiRdr.version = common.STI_V2

		plainHdrBody := &searchidxv1.StiPlainHdrBodyV1{}
		plainHdrBody, err := plainHdrBody.Deserialize(plainHdrBodyData)
		if err != nil {
			return errors.New(err)
		}

		stiv2, err := searchidxv2.OpenSearchTermIdxPrivV2(owner, termID, updateID, termKey,
			indexKey, reader, plainHdr, 0, uint64(plainHdrSize), size)
		if err != nil {
			return err
		}

		stiRdrAnalyzer, err := tokenizer.OpenAnalyzer(stiv2.TokenizerType)
		if err != nil {
			return err
		}

		stiRdr.analyzer = stiRdrAnalyzer
		stiRdr.stiV2 = stiv2
	default:
		return errors.Errorf("Search term index version %v is not supported",
			version.GetStiVersion())
	}

	analzdTerms, origToAnalzd, analzdToOrigs, err := common.AnalyzeTerms(origTerms, stiRdr.analyzer)
	if err != nil {
		return err
	}

	stiRdr.analzdTerms = analzdTerms
	stiRdr.origToAnalzd = origToAnalzd
	stiRdr.analzdToOrigs = analzdToOrigs

	bReader.stiReaders = append(bReader.stiReaders, stiRdr)

	return nil
}

type stiChanResult struct {
	data *stiData
	err  error
}

func (bReader *stiBulkReader) ReadNextData() (StiData, error) {
	finalData := &stiData{
		termDoc: make(map[string]StiDocOffsets), // term -> stiTermDoc
		termEof: make(map[string]bool),          // term -> eof(bool)
	}

	if len(bReader.stiReaders) == 0 {
		return nil, io.EOF
	}

	readerToChan := make(map[*stiReader](chan *stiChanResult))
	for _, stiRdr := range bReader.stiReaders {
		channel := make(chan *stiChanResult)
		readerToChan[stiRdr] = channel

		// This executes in a separate thread
		go bReader.readStiData(stiRdr, channel)
	}

	allEOF := true
	for _, channel := range readerToChan {
		result := <-channel
		if result.err != nil {
			if result.err != io.EOF {
				return nil, result.err
			}
		} else {
			allEOF = false // There are some data
		}

		if result.data != nil {
			for term, termDoc := range result.data.termDoc {
				finalData.termDoc[term] = termDoc
			}
			for term, termEof := range result.data.termEof {
				finalData.termEof[term] = termEof
			}
		}
	}

	if allEOF {
		return finalData, io.EOF
	}

	return finalData, nil
}

func (bReader *stiBulkReader) readStiData(reader *stiReader, channel chan<- *stiChanResult) {
	defer close(channel)
	result := &stiChanResult{
		data: nil,
		err:  nil,
	}

	if reader == nil {
		result.err = io.EOF
		channel <- result
		return
	}

	switch reader.version {
	case common.STI_V1:
		blk, err := reader.stiV1.ReadNextBlock()
		if err != nil {
			result.err = err
			if err != io.EOF {
				channel <- result
				return
			}
		}

		result.data = &stiData{
			termDoc: make(map[string]StiDocOffsets), // term -> stiTermDoc
			termEof: make(map[string]bool),          // term -> eof(bool)
		}

		for _, term := range reader.analzdTerms {
			result.data.termEof[term] = (err == io.EOF)
		}

		if blk == nil {
			channel <- result
			return
		}

		term := reader.analzdTerms[0]
		termDoc := &stiTermDoc{
			docIDs:     make([]string, 0, len(blk.DocVerOffset)),
			docOffsets: make(map[string]*stiDocOffsets), // docID -> stiDocOffsets
		}
		result.data.termDoc[term] = termDoc

		for docID, verOffset := range blk.DocVerOffset {
			termDoc.docIDs = append(termDoc.docIDs, docID)
			docOffset := termDoc.docOffsets[docID]
			if docOffset == nil {
				docOffset = &stiDocOffsets{
					docID:   docID,
					docVer:  verOffset.Version,
					offsets: make([]uint64, 0, len(verOffset.Offsets)),
				}
				termDoc.docOffsets[docID] = docOffset
			}
			docOffset.offsets = append(docOffset.offsets, verOffset.Offsets...)
		}

	case common.STI_V2:
		blk, err := reader.stiV2.ReadNextBlock()
		if err != nil {
			result.err = err
			if err != io.EOF {
				channel <- result
				return
			}
		}

		result.data = &stiData{
			termDoc: make(map[string]StiDocOffsets), // term -> stiTermDoc
			termEof: make(map[string]bool),          // term -> eof(bool)
		}

		existTerm := false
		for _, term := range reader.analzdTerms {
			result.data.termEof[term] = (err == io.EOF)
			for _, origTerms := range reader.analzdToOrigs[term] {
				result.data.termEof[origTerms] = result.data.termEof[term]
			}
			if blk != nil {
				if _, ok := blk.TermDocVerOffset[term]; ok {
					existTerm = true
				}
			}
		}

		if blk == nil && !existTerm {
			channel <- result
			return
		}

		termToStiTermDoc := make(map[string]*stiTermDoc)
		for term, docIDVerOffset := range blk.TermDocVerOffset {
			if _, ok := termToStiTermDoc[term]; !ok {
				termToStiTermDoc[term] = &stiTermDoc{
					docIDs:     make([]string, 0),
					docOffsets: make(map[string]*stiDocOffsets),
				}
				result.data.termDoc[term] = termToStiTermDoc[term]
				for _, origTerms := range reader.analzdToOrigs[term] {
					result.data.termDoc[origTerms] = termToStiTermDoc[term]
				}
			}
			termDoc := termToStiTermDoc[term]
			for docID, verOffset := range docIDVerOffset {
				termDoc.docIDs = append(termDoc.docIDs, docID)
				docOffset := termDoc.docOffsets[docID]
				if docOffset == nil {
					docOffset = &stiDocOffsets{
						docID:   docID,
						docVer:  verOffset.Version,
						offsets: make([]uint64, 0, len(verOffset.Offsets)),
					}
					termDoc.docOffsets[docID] = docOffset
				}
				docOffset.offsets = append(docOffset.offsets, verOffset.Offsets...)
			}
		}
	default:
		result.err = errors.Errorf("Search term index version %v is not supported",
			reader.version)
	}

	channel <- result
}

func (bReader *stiBulkReader) GetTerms() []string {
	return bReader.analzdTerms
}

func (bReader *stiBulkReader) Reset() error {
	var err error = nil

	for _, stiRdr := range bReader.stiReaders {
		if stiRdr != nil {
			switch stiRdr.version {
			case common.STI_V1:
				err = utils.FirstError(err, stiRdr.stiV1.Reset())
			case common.STI_V2:
				err = utils.FirstError(err, stiRdr.stiV2.Reset())
			default:
				return errors.Errorf("Search term index version %v is not supported",
					stiRdr.version)
			}
		}
	}

	return err
}

func (bReader *stiBulkReader) Close() error {
	var err error = nil

	for _, stiRdr := range bReader.stiReaders {
		if stiRdr != nil {
			switch stiRdr.version {
			case common.STI_V1:
				err = utils.FirstError(err, stiRdr.stiV1.Close())
			case common.STI_V2:
				err = utils.FirstError(err, stiRdr.stiV2.Close())
			default:
				return errors.Errorf("Search term index version %v is not supported",
					stiRdr.version)
			}
		}
	}

	return err
}

func (data *stiData) IsTermEOF(term string) bool {
	return data.termEof[term]
}

func (data *stiData) GetTermDocOffsets(term string) StiDocOffsets {
	return data.termDoc[term]
}

func (data *stiData) GetTermsDocOffsets(terms []string) map[string]StiDocOffsets {
	termsDocOffsets := make(map[string]StiDocOffsets)
	for _, term := range terms {
		termsDocOffsets[term] = data.termDoc[term]
	}
	return termsDocOffsets
}

func (data *stiData) GetAllDocOffsets() map[string]StiDocOffsets {
	return data.termDoc
}

func (tdo *stiTermDoc) GetDocIDs() []string {
	return tdo.docIDs
}

func (tdo *stiTermDoc) GetDocVer(docID string) uint64 {
	docOffset := tdo.docOffsets[docID]
	if docOffset == nil {
		return 0
	}
	return docOffset.docVer
}

func (tdo *stiTermDoc) GetOffsets(docID string) []uint64 {
	docOffset := tdo.docOffsets[docID]
	if docOffset == nil {
		return nil
	}
	return docOffset.offsets
}

func (tdo *stiTermDoc) GetDocIdOffsets() map[string][]uint64 {
	docIdOffsetMap := make(map[string][]uint64) // DocID -> Offsets
	for docID, offsets := range tdo.docOffsets {
		if offsets != nil {
			docIdOffsetMap[docID] = append(docIdOffsetMap[docID], offsets.offsets...)
		}
	}

	return docIdOffsetMap
}

//////////////////////////////////////////////////////////////////
//
//                    Search Sorted Doc Index
//
//////////////////////////////////////////////////////////////////

type SsdiDoc interface {
	GetDocID() string
	GetDocVer() uint64
}

type SsdiData interface {
	GetTermDocs() map[string][]SsdiDoc // term -> []SsdiDoc
	GetDocs(term string) []SsdiDoc
	GetDoc(term string, docID string) SsdiDoc
	GetDocVer(term string, docID string, docVer uint64) SsdiDoc
}

type SsdiReader interface {
	ReadNextData() (SsdiData, error)
	Reset() error
	Close() error
}

type ssdiDoc struct {
	docID  string
	docVer uint64
}

type ssdiData struct {
	termDoc   map[string][]SsdiDoc          // term -> []ssdiDoc
	termDocID map[string]map[string]SsdiDoc // term -> (docID -> ssdiDoc)
}

type ssdiReader struct {
	reader        io.ReadCloser
	size          uint64
	version       uint32
	termID        string
	analzdTerms   []string            // analyzed terms
	origTerms     []string            // original terms
	origToAnalzd  map[string]string   // original term -> analyzed term
	analzdToOrigs map[string][]string // analyzed term -> []original term
	analyzer      tokenizer.Analyzer
	ssdiV1        *searchidxv1.SearchSortDocIdxV1
	ssdiV2        *searchidxv2.SearchSortDocIdxV2
}

type ssdiBulkReader struct {
	analzdTerms []string
	ssdiReaders []*ssdiReader
	analyzer    tokenizer.Analyzer
}

func OpenSearchSortedDocIndex(sdc client.StrongDocClient, owner common.SearchIdxOwner, origTerms []string,
	termKey, indexKey *sscrypto.StrongSaltKey, ssdiVersion uint32) (SsdiReader, error) {

	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The term key type %v is not a MAC key", termKey.Type.Name)
	}

	if indexKey.Type != sscrypto.Type_XChaCha20 {
		return nil, errors.Errorf("Index key type %v is not supported. The only supported key type is %v",
			indexKey.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	analyzer, err := GetSearchTermAnalyzer()
	if err != nil {
		return nil, err
	}

	analzdTerms, _, analzdToOrigs, err := common.AnalyzeTerms(origTerms, analyzer)
	if err != nil {
		return nil, err
	}

	// termID -> list of terms
	termIDMap, err := common.GetTermIDs(analzdTerms, termKey, common.STI_TERM_BUCKET_COUNT, ssdiVersion)
	if err != nil {
		return nil, err
	}

	bulkReader := &ssdiBulkReader{
		analzdTerms: analzdTerms,
		ssdiReaders: make([]*ssdiReader, 0, len(termIDMap)),
		analyzer:    analyzer,
	}

	for termID, analyzedTerms := range termIDMap {
		var readerOrigTerms []string = nil
		for _, analyzedTerm := range analyzedTerms {
			readerOrigTerms = append(readerOrigTerms, analzdToOrigs[analyzedTerm]...)
		}
		err := bulkReader.initSsdiReader(sdc, owner, termID, readerOrigTerms,
			termKey, indexKey, ssdiVersion)
		if err != nil && err != os.ErrNotExist {
			return nil, err
		}
	} // for termID, analyzedTerms := range termIDMap

	return bulkReader, nil
}

func (bReader *ssdiBulkReader) initSsdiReader(sdc client.StrongDocClient, owner common.SearchIdxOwner,
	termID string, origTerms []string, termKey, indexKey *sscrypto.StrongSaltKey, ssdiVersion uint32) error {
	updateID, err := common.GetLatestUpdateID(sdc, owner, termID)
	if err != nil {
		return err
	}

	reader, size, err := common.OpenSearchSortDocIndexReader(sdc, owner, termID, updateID)
	if err != nil {
		return err
	}

	ssdiRdr := &ssdiReader{
		reader:        reader,
		size:          size,
		version:       ssdiVersion,
		termID:        termID,
		analzdTerms:   nil,
		origTerms:     origTerms,
		origToAnalzd:  make(map[string]string),
		analzdToOrigs: make(map[string][]string),
	}

	plainHdr, plainHdrSize, err := ssheaders.DeserializePlainHdrStream(reader)
	if err != nil {
		return err
	}

	plainHdrBodyData, err := plainHdr.GetBody()
	if err != nil {
		return err
	}

	version, err := common.DeserializeSsdiVersion(plainHdrBodyData)
	if err != nil {
		return err
	}

	switch version.GetSsdiVersion() {
	case common.SSDI_V1:
		ssdiRdr.version = common.SSDI_V1

		// Parse plaintext header body
		plainHdrBody := &searchidxv1.StiPlainHdrBodyV1{}
		plainHdrBody, err := plainHdrBody.Deserialize(plainHdrBodyData)
		if err != nil {
			return errors.New(err)
		}

		ssdiRdrAnalyzer, err := GetSearchTermAnalyzer()
		if err != nil {
			return err
		}

		analyzedTerms := ssdiRdrAnalyzer.Analyze(origTerms[0])
		ssdiv1, err := searchidxv1.OpenSearchSortDocIdxPrivV1(owner, analyzedTerms[0], termID, updateID,
			termKey, indexKey, reader, plainHdr, 0, uint64(plainHdrSize), size)
		if err != nil {
			return err
		}

		ssdiRdr.analyzer = ssdiRdrAnalyzer
		ssdiRdr.ssdiV1 = ssdiv1
	case common.SSDI_V2:
		ssdiRdr.version = common.SSDI_V2

		// Parse plaintext header body
		plainHdrBody := &searchidxv1.StiPlainHdrBodyV1{}
		plainHdrBody, err := plainHdrBody.Deserialize(plainHdrBodyData)
		if err != nil {
			return errors.New(err)
		}

		ssdiv2, err := searchidxv2.OpenSearchSortDocIdxPrivV2(owner, termID, updateID,
			termKey, indexKey, reader, plainHdr, 0, uint64(plainHdrSize), size)
		if err != nil {
			return err
		}

		ssdiRdrAnalyzer, err := tokenizer.OpenAnalyzer(ssdiv2.TokenizerType)
		if err != nil {
			return err
		}

		ssdiRdr.analyzer = ssdiRdrAnalyzer
		ssdiRdr.ssdiV2 = ssdiv2
	default:
		return errors.Errorf("Search sorted document index version %v is not supported",
			version.GetSsdiVersion())
	}

	analzdTerms, origToAnalzd, analzdToOrigs, err := common.AnalyzeTerms(origTerms, ssdiRdr.analyzer)
	if err != nil {
		return err
	}

	ssdiRdr.analzdTerms = analzdTerms
	ssdiRdr.origToAnalzd = origToAnalzd
	ssdiRdr.analzdToOrigs = analzdToOrigs

	bReader.ssdiReaders = append(bReader.ssdiReaders, ssdiRdr)

	return nil
}

type ssdiChanResult struct {
	data *ssdiData
	err  error
}

func (bReader *ssdiBulkReader) ReadNextData() (SsdiData, error) {
	finalData := &ssdiData{
		termDoc:   make(map[string][]SsdiDoc),          // term -> []SsdiDoc
		termDocID: make(map[string]map[string]SsdiDoc), // term -> (docID -> ssdiDoc)
	}

	if len(bReader.ssdiReaders) == 0 {
		return nil, io.EOF
	}

	readerToChan := make(map[*ssdiReader](chan *ssdiChanResult))
	for _, ssdiRdr := range bReader.ssdiReaders {
		channel := make(chan *ssdiChanResult)
		readerToChan[ssdiRdr] = channel

		// This executes in a separate thread
		go bReader.readSsdiData(ssdiRdr, channel)
	}

	allEOF := true
	for _, channel := range readerToChan {
		result := <-channel
		if result.err != nil {
			if result.err != io.EOF {
				return nil, result.err
			}
		} else {
			allEOF = false
		}

		if result.data != nil {
			for term, docs := range result.data.termDoc {
				finalData.termDoc[term] = docs
			}
			for term, docIDMap := range result.data.termDocID {
				finalData.termDocID[term] = docIDMap
			}
		}
	}

	if allEOF {
		return finalData, io.EOF
	}

	return finalData, nil
}

func (bReader *ssdiBulkReader) readSsdiData(reader *ssdiReader, channel chan<- *ssdiChanResult) {
	defer close(channel)
	result := &ssdiChanResult{
		data: &ssdiData{
			termDoc:   make(map[string][]SsdiDoc),          // term -> []SsdiDoc
			termDocID: make(map[string]map[string]SsdiDoc), // term -> (docID -> ssdiDoc)
		},
		err: nil,
	}

	if reader == nil {
		result.err = io.EOF
		channel <- result
		return
	}

	switch reader.version {
	case common.SSDI_V1:
		blk, err := reader.ssdiV1.ReadNextBlock()
		if err != nil {
			result.err = err
			if err != io.EOF {
				channel <- result
				return
			}
		}

		if blk == nil {
			channel <- result
			return
		}

		term := reader.analzdTerms[0]
		docs := make([]SsdiDoc, 0, len(blk.DocIDVers))
		docIDMap := make(map[string]SsdiDoc)
		for _, doc := range blk.DocIDVers {
			ssdiDoc := &ssdiDoc{doc.DocID, doc.DocVer}
			docs = append(docs, ssdiDoc)
			docIDMap[doc.DocID] = ssdiDoc
		}
		result.data.termDoc[term] = docs
		result.data.termDocID[term] = docIDMap
	case common.SSDI_V2:
		blk, err := reader.ssdiV2.ReadNextBlock()
		if err != nil {
			result.err = err
			if err != io.EOF {
				channel <- result
				return
			}
		}

		if blk == nil {
			channel <- result
			return
		}

		for _, termDocIDVer := range blk.TermDocIDVers {
			term := termDocIDVer.Term
			docID := termDocIDVer.DocID
			docVer := termDocIDVer.DocVer
			if _, exists := reader.analzdToOrigs[term]; !exists {
				continue
			}

			ssdiDoc := &ssdiDoc{docID: docID, docVer: docVer}

			// update result.data.termDoc
			result.data.termDoc[term] = append(result.data.termDoc[term], ssdiDoc)

			// update result.data.termDocID
			if _, exists := result.data.termDocID[term]; !exists {
				result.data.termDocID[term] = make(map[string]SsdiDoc)
			}
			result.data.termDocID[term][docID] = ssdiDoc
		}

		for term := range result.data.termDoc {
			for _, origTerm := range reader.analzdToOrigs[term] {
				result.data.termDoc[origTerm] = result.data.termDoc[term]
				result.data.termDocID[origTerm] = result.data.termDocID[term]
			}
		}
	default:
		result.err = errors.Errorf("Search sorted document version %v is not supported",
			reader.version)
	}

	channel <- result
}

func (bReader *ssdiBulkReader) Reset() error {
	var err error = nil

	for _, ssdiRdr := range bReader.ssdiReaders {
		if ssdiRdr != nil {
			switch ssdiRdr.version {
			case common.STI_V1:
				err = utils.FirstError(err, ssdiRdr.ssdiV1.Reset())
			default:
				return errors.Errorf("Search sorted document index version %v is not supported",
					ssdiRdr.version)
			}
		}
	}

	return err
}

func (bReader *ssdiBulkReader) Close() error {
	var err error = nil

	for _, ssdiRdr := range bReader.ssdiReaders {
		if ssdiRdr != nil {
			switch ssdiRdr.version {
			case common.STI_V1:
				err = utils.FirstError(err, ssdiRdr.ssdiV1.Close())
			default:
				return errors.Errorf("Search sorted document index version %v is not supported",
					ssdiRdr.version)
			}
		}
	}

	return err
}

func (data *ssdiData) GetTermDocs() map[string][]SsdiDoc { // term -> []SsdiDoc
	return data.termDoc
}

func (data *ssdiData) GetDocs(term string) []SsdiDoc {
	return data.termDoc[term]
}

func (data *ssdiData) GetDoc(term string, docID string) SsdiDoc {
	docIDMap := data.termDocID[term]
	if docIDMap == nil {
		return nil
	}
	return docIDMap[docID]
}

func (data *ssdiData) GetDocVer(term string, docID string, docVer uint64) SsdiDoc {
	doc := data.GetDoc(term, docID)
	if doc != nil && doc.GetDocVer() == docVer {
		return doc
	}
	return nil
}

func (doc *ssdiDoc) GetDocID() string {
	return doc.docID
}

func (doc *ssdiDoc) GetDocVer() uint64 {
	return doc.docVer
}
