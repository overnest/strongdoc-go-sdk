package searchidx

import (
	"io"
	"os"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/searchidxv1"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/searchidxv2"
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
	termDoc map[string]StiDocOffsets // term -> stiTermDoc
	termEof map[string]bool          // term -> eof(bool)
}

type stiReader struct {
	reader  io.ReadCloser
	size    uint64
	version uint32
	termID  string
	terms   []string
	termMap map[string]bool
	stiV1   *searchidxv1.SearchTermIdxV1
	stiV2   *searchidxv2.SearchTermIdxV2
}

type stiBulkReader struct {
	terms         []string
	stiReaders    []*stiReader
	termInfoMap   map[string]*stiReader
	termIdInfoMap map[string]*stiReader
}

func OpenSearchTermIndex(sdc client.StrongDocClient, owner common.SearchIdxOwner, terms []string,
	termKey, indexKey *sscrypto.StrongSaltKey, stiVersion uint32) (StiReader, error) {

	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The term key type %v is not a MAC key", termKey.Type.Name)
	}

	if indexKey.Type != sscrypto.Type_XChaCha20 {
		return nil, errors.Errorf("Index key type %v is not supported. The only supported key type is %v",
			indexKey.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	// termID -> list of terms
	termIDMap, err := common.GetTermIDs(terms, termKey, common.STI_TERM_BUCKET_COUNT, stiVersion)
	if err != nil {
		return nil, err
	}

	bReader := &stiBulkReader{
		terms:         terms,
		stiReaders:    make([]*stiReader, 0, len(termIDMap)),
		termInfoMap:   make(map[string]*stiReader),
		termIdInfoMap: make(map[string]*stiReader),
	}

	for termID, terms := range termIDMap {
		updateID, err := common.GetLatestUpdateID(sdc, owner, termID)
		if err != nil && err != os.ErrNotExist {
			return nil, err
		}

		var size uint64 = 0
		var reader io.ReadCloser = nil
		if err == nil {
			reader, size, err = common.OpenSearchTermIndexReader(sdc, owner, termID, updateID)
			if err != nil && err != os.ErrNotExist {
				return nil, err
			}
		}

		if err == os.ErrNotExist {
			bReader.termIdInfoMap[termID] = nil
			for _, term := range terms {
				bReader.termInfoMap[term] = nil
			}
			continue
		}

		stiRdr := &stiReader{
			reader:  reader,
			size:    size,
			version: stiVersion,
			termID:  termID,
			terms:   terms,
			termMap: make(map[string]bool),
		}

		for _, term := range terms {
			stiRdr.termMap[term] = true
		}

		plainHdr, plainHdrSize, err := ssheaders.DeserializePlainHdrStream(reader)
		if err != nil {
			return nil, err
		}

		plainHdrBodyData, err := plainHdr.GetBody()
		if err != nil {
			return nil, err
		}

		version, err := common.DeserializeStiVersion(plainHdrBodyData)
		if err != nil {
			return nil, err
		}

		switch version.GetStiVersion() {
		case common.STI_V1:
			stiRdr.version = common.STI_V1

			// Parse plaintext header body
			plainHdrBody := &searchidxv1.StiPlainHdrBodyV1{}
			plainHdrBody, err := plainHdrBody.Deserialize(plainHdrBodyData)
			if err != nil {
				return nil, errors.New(err)
			}

			stiv1, err := searchidxv1.OpenSearchTermIdxPrivV1(owner, terms[0], termID, updateID, termKey,
				indexKey, reader, plainHdr, 0, uint64(plainHdrSize), size)
			if err != nil {
				return nil, err
			}

			stiRdr.stiV1 = stiv1

		case common.STI_V2:
			stiRdr.version = common.STI_V2

			plainHdrBody := &searchidxv1.StiPlainHdrBodyV1{}
			plainHdrBody, err := plainHdrBody.Deserialize(plainHdrBodyData)
			if err != nil {
				return nil, errors.New(err)
			}

			stiv2, err := searchidxv2.OpenSearchTermIdxPrivV2(owner, termID, updateID, termKey,
				indexKey, reader, plainHdr, 0, uint64(plainHdrSize), size)
			if err != nil {
				return nil, err
			}

			stiRdr.stiV2 = stiv2

		default:
			return nil, errors.Errorf("Search term index version %v is not supported",
				version.GetStiVersion())
		}

		bReader.stiReaders = append(bReader.stiReaders, stiRdr)
		bReader.termIdInfoMap[termID] = stiRdr
	} // for termID, terms := range termIDMap

	return bReader, nil
}

func (reader *stiBulkReader) ReadNextData() (StiData, error) {
	finalData := &stiData{
		termDoc: make(map[string]StiDocOffsets), // term -> stiTermDoc
		termEof: make(map[string]bool),          // term -> eof(bool)
	}

	if len(reader.stiReaders) == 0 {
		return nil, io.EOF
	}

	type chanResult struct {
		data *stiData
		err  error
	}

	readerToChan := make(map[*stiReader](chan *chanResult))
	for _, stiRdr := range reader.stiReaders {
		channel := make(chan *chanResult)
		readerToChan[stiRdr] = channel

		// This executes in a separate thread
		go func(reader *stiReader, channel chan<- *chanResult) {
			defer close(channel)
			result := &chanResult{
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

				for _, term := range reader.terms {
					result.data.termEof[term] = (err == io.EOF)
				}

				if blk == nil {
					channel <- result
					return
				}

				// TODO: May need to change later
				term := reader.terms[0]
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
				for _, term := range reader.terms {
					result.data.termEof[term] = (err == io.EOF)
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
		}(stiRdr, channel)
	} // for _, stiRdr := range reader.stiReaders

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

func (reader *stiBulkReader) GetTerms() []string {
	return reader.terms
}

func (reader *stiBulkReader) Reset() error {
	var err error = nil

	for _, stiRdr := range reader.stiReaders {
		if stiRdr != nil {
			switch stiRdr.version {
			case common.STI_V1:
				err = utils.FirstError(err, stiRdr.stiV1.Reset())
			default:
				return errors.Errorf("Search term index version %v is not supported",
					stiRdr.version)
			}
		}
	}

	return err
}

func (reader *stiBulkReader) Close() error {
	var err error = nil

	for _, stiRdr := range reader.stiReaders {
		if stiRdr != nil {
			switch stiRdr.version {
			case common.STI_V1:
				err = utils.FirstError(err, stiRdr.stiV1.Close())
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
	reader  io.ReadCloser
	size    uint64
	version uint32
	termID  string
	terms   []string
	termMap map[string]bool
	ssdiV1  *searchidxv1.SearchSortDocIdxV1
	ssdiV2  *searchidxv2.SearchSortDocIdxV2
}

type ssdiBulkReader struct {
	terms         []string
	ssdiReaders   []*ssdiReader
	termInfoMap   map[string]*ssdiReader // term -> ssdiReader
	termIdInfoMap map[string]*ssdiReader // termID -> ssdiReader
}

func OpenSearchSortedDocIndex(sdc client.StrongDocClient, owner common.SearchIdxOwner, terms []string,
	termKey, indexKey *sscrypto.StrongSaltKey, ssdiVersion uint32) (SsdiReader, error) {

	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The term key type %v is not a MAC key", termKey.Type.Name)
	}

	if indexKey.Type != sscrypto.Type_XChaCha20 {
		return nil, errors.Errorf("Index key type %v is not supported. The only supported key type is %v",
			indexKey.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	// termID -> list of terms
	termIDMap, err := common.GetTermIDs(terms, termKey, common.STI_TERM_BUCKET_COUNT, ssdiVersion)
	if err != nil {
		return nil, err
	}

	bReader := &ssdiBulkReader{
		terms:         terms,
		ssdiReaders:   make([]*ssdiReader, 0, len(termIDMap)),
		termInfoMap:   make(map[string]*ssdiReader),
		termIdInfoMap: make(map[string]*ssdiReader),
	}

	for termID, terms := range termIDMap {
		updateID, err := common.GetLatestUpdateID(sdc, owner, termID)
		if err != nil && err != os.ErrNotExist {
			return nil, err
		}

		var size uint64 = 0
		var reader io.ReadCloser = nil
		if err == nil {
			reader, size, err = common.OpenSearchSortDocIndexReader(sdc, owner, termID, updateID)
			if err != nil && err != os.ErrNotExist {
				return nil, err
			}
		}

		if err == os.ErrNotExist {
			bReader.termIdInfoMap[termID] = nil
			for _, term := range terms {
				bReader.termInfoMap[term] = nil
			}
			continue
		}

		ssdiRdr := &ssdiReader{
			reader:  reader,
			size:    size,
			version: ssdiVersion,
			termID:  termID,
			terms:   terms,
			termMap: make(map[string]bool),
		}

		for _, term := range terms {
			ssdiRdr.termMap[term] = true
		}

		plainHdr, plainHdrSize, err := ssheaders.DeserializePlainHdrStream(reader)
		if err != nil {
			return nil, err
		}

		plainHdrBodyData, err := plainHdr.GetBody()
		if err != nil {
			return nil, err
		}

		version, err := common.DeserializeSsdiVersion(plainHdrBodyData)
		if err != nil {
			return nil, err
		}

		switch version.GetSsdiVersion() {
		case common.SSDI_V1:
			ssdiRdr.version = common.SSDI_V1

			// Parse plaintext header body
			plainHdrBody := &searchidxv1.StiPlainHdrBodyV1{}
			plainHdrBody, err := plainHdrBody.Deserialize(plainHdrBodyData)
			if err != nil {
				return nil, errors.New(err)
			}

			ssdiv1, err := searchidxv1.OpenSearchSortDocIdxPrivV1(owner, terms[0], termID, updateID,
				termKey, indexKey, reader, plainHdr, 0, uint64(plainHdrSize), size)
			if err != nil {
				return nil, err
			}

			ssdiRdr.ssdiV1 = ssdiv1
		case common.SSDI_V2:
			ssdiRdr.version = common.SSDI_V2

			// Parse plaintext header body
			plainHdrBody := &searchidxv1.StiPlainHdrBodyV1{}
			plainHdrBody, err := plainHdrBody.Deserialize(plainHdrBodyData)
			if err != nil {
				return nil, errors.New(err)
			}

			ssdiv2, err := searchidxv2.OpenSearchSortDocIdxPrivV2(owner, termID, updateID,
				termKey, indexKey, reader, plainHdr, 0, uint64(plainHdrSize), size)
			if err != nil {
				return nil, err
			}

			ssdiRdr.ssdiV2 = ssdiv2
		default:
			return nil, errors.Errorf("Search sorted document index version %v is not supported",
				version.GetSsdiVersion())
		}

		bReader.ssdiReaders = append(bReader.ssdiReaders, ssdiRdr)
		bReader.termIdInfoMap[termID] = ssdiRdr
	} // for termID, terms := range termIDMap

	return bReader, nil
}

func (reader *ssdiBulkReader) ReadNextData() (SsdiData, error) {
	finalData := &ssdiData{
		termDoc:   make(map[string][]SsdiDoc),          // term -> []SsdiDoc
		termDocID: make(map[string]map[string]SsdiDoc), // term -> (docID -> ssdiDoc)
	}

	if len(reader.ssdiReaders) == 0 {
		return nil, io.EOF
	}

	type chanResult struct {
		data *ssdiData
		err  error
	}

	readerToChan := make(map[*ssdiReader](chan *chanResult))
	for _, ssdiRdr := range reader.ssdiReaders {
		channel := make(chan *chanResult)
		readerToChan[ssdiRdr] = channel

		// This executes in a separate thread
		go func(reader *ssdiReader, channel chan<- *chanResult) {
			defer close(channel)
			result := &chanResult{
				data: nil,
				err:  nil,
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

				data := &ssdiData{
					termDoc:   make(map[string][]SsdiDoc),          // term -> []SddiDoc
					termDocID: make(map[string]map[string]SsdiDoc), // term -> (docID -> ssdiDoc)
				}
				result.data = data

				// TODO: May need to change later
				term := reader.terms[0]
				docs := make([]SsdiDoc, 0, len(blk.DocIDVers))
				docIDMap := make(map[string]SsdiDoc)
				for _, doc := range blk.DocIDVers {
					ssdiDoc := &ssdiDoc{doc.DocID, doc.DocVer}
					docs = append(docs, ssdiDoc)
					docIDMap[doc.DocID] = ssdiDoc
				}
				data.termDoc[term] = docs
				data.termDocID[term] = docIDMap
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

				data := &ssdiData{
					termDoc:   make(map[string][]SsdiDoc),          // term -> []SddiDoc
					termDocID: make(map[string]map[string]SsdiDoc), // term -> (docID -> ssdiDoc)
				}
				result.data = data

				readerTerms := make(map[string]bool)
				for _, term := range reader.terms {
					readerTerms[term] = true
				}

				for _, termDocIDVer := range blk.TermDocIDVers {
					term := termDocIDVer.Term
					docID := termDocIDVer.DocID
					docVer := termDocIDVer.DocVer
					if _, exists := readerTerms[term]; !exists {
						continue
					}

					ssdiDoc := &ssdiDoc{docID: docID, docVer: docVer}

					// update data.termDoc
					if _, exists := data.termDoc[term]; !exists {
						data.termDoc[term] = []SsdiDoc{}
					}
					data.termDoc[term] = append(data.termDoc[term], ssdiDoc)

					// update data.termDocID
					if _, exists := data.termDocID[term]; !exists {
						data.termDocID[term] = make(map[string]SsdiDoc)
					}
					data.termDocID[term][docID] = ssdiDoc
				}

			default:
				result.err = errors.Errorf("Search sorted document version %v is not supported",
					reader.version)
			}

			channel <- result
		}(ssdiRdr, channel)
	} // for _, stiRdr := range reader.stiReaders

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

func (reader *ssdiBulkReader) Reset() error {
	var err error = nil

	for _, ssdiRdr := range reader.ssdiReaders {
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

func (reader *ssdiBulkReader) Close() error {
	var err error = nil

	for _, ssdiRdr := range reader.ssdiReaders {
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
