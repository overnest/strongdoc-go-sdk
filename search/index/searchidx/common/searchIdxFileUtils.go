package common

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/utils"
)

func newUpdateIDV1() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

// GetSearchIdxPathPrefix gets the search index path prefix
// return /tmp/search/<owner>/sidx
func GetSearchIdxPathPrefix(owner SearchIdxOwner) string {
	return fmt.Sprintf("/tmp/search/%v/sidx", owner)
}

// getSearchIdxPath gets the base path of the search index
// return /tmp/search/<owner>/sidx/<term>/updateID
func getSearchIdxPath(owner SearchIdxOwner, term, updateID string) string {
	return fmt.Sprintf("/%v/%v/%v", GetSearchIdxPathPrefix(owner), term, updateID)
}

//  return /tmp/search/<owner>/sidx/<term>/updateID/searchterm
func getSearchTermIdxPath(owner SearchIdxOwner, term, updateID string) string {
	return fmt.Sprintf("%v/searchterm", getSearchIdxPath(owner, term, updateID))
}

//  return /tmp/search/<owner>/sidx/<term>/updateID/sortdoc
func getSearchSortDocIdxPath(owner SearchIdxOwner, term, updateID string) string {
	return fmt.Sprintf("%v/sortdoc", getSearchIdxPath(owner, term, updateID))
}

func OpenSearchTermIndexReader(sdc client.StrongDocClient, owner SearchIdxOwner, termHmac string, updateID string) (io.ReadCloser, uint64, error) {
	if utils.TestLocal {
		path := getSearchTermIdxPath(owner, termHmac, updateID)
		reader, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, 0, os.ErrNotExist
			}
			return nil, 0, err
		}

		stat, err := reader.Stat()
		if err != nil {
			return nil, 0, err
		}

		return reader, uint64(stat.Size()), nil

	}
	reader, err := api.NewSearchTermIdxReader(sdc, owner.GetOwnerType(), termHmac, updateID)
	if err != nil {
		return nil, 0, err
	}

	size, err := api.GetSearchTermIndexSize(sdc, owner.GetOwnerType(), termHmac, updateID)
	if err != nil {
		return nil, 0, err
	}
	return reader, size, nil
}

func OpenSearchSortDocIndexReader(sdc client.StrongDocClient, owner SearchIdxOwner, termHmac string, updateID string) (io.ReadCloser, uint64, error) {
	if utils.TestLocal {
		path := getSearchSortDocIdxPath(owner, termHmac, updateID)
		reader, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, 0, os.ErrNotExist
			}
			return nil, 0, err
		}

		stat, err := reader.Stat()
		if err != nil {
			return nil, 0, err
		}

		return reader, uint64(stat.Size()), nil

	}
	reader, err := api.NewSearchSortedDocIdxReader(sdc, owner.GetOwnerType(), termHmac, updateID)
	if err != nil {
		return nil, 0, err
	}

	size, err := api.GetSearchSortedDocIdxSize(sdc, owner.GetOwnerType(), termHmac, updateID)
	if err != nil {
		return nil, 0, err
	}
	return reader, size, nil
}

func OpenSearchSortDocIndexWriter(sdc client.StrongDocClient, owner SearchIdxOwner, termHmac string, updateID string) (io.WriteCloser, error) {
	if utils.TestLocal {
		path := getSearchSortDocIdxPath(owner, termHmac, updateID)
		return utils.MakeDirAndCreateFile(path)
	}
	return api.NewSearchSortedDocIdxWriter(sdc, owner.GetOwnerType(), termHmac, updateID)
}

func OpenSearchTermIndexWriter(sdc client.StrongDocClient, owner SearchIdxOwner, termHmac string) (writer io.WriteCloser, updateID string, err error) {
	if utils.TestLocal {
		updateID = newUpdateIDV1()
		path := getSearchTermIdxPath(owner, termHmac, updateID)
		writer, err = utils.MakeDirAndCreateFile(path)
		return
	}
	return api.NewSearchTermIdxWriter(sdc, owner.GetOwnerType(), termHmac)
}

func GetUpdateIDs(sdc client.StrongDocClient, owner SearchIdxOwner, termHmac string) ([]string, error) {
	if utils.TestLocal {
		path := getSearchIdxPath(owner, termHmac, "")

		files, err := ioutil.ReadDir(path)
		if err != nil {
			return nil, err
		}

		updateIDs := make([]string, 0, len(files))
		for _, file := range files {
			if file.IsDir() {
				updateIDs = append(updateIDs, file.Name())
			}
		}

		updateIDsInt := make([]int64, len(updateIDs))
		for i := 0; i < len(updateIDs); i++ {
			id, err := strconv.ParseInt(updateIDs[i], 16, 64)
			if err == nil {
				updateIDsInt[i] = id
			}
		}

		sort.Slice(updateIDsInt, func(i, j int) bool { return updateIDsInt[i] > updateIDsInt[j] })
		for i := 0; i < len(updateIDs); i++ {
			updateIDs[i] = fmt.Sprintf("%x", updateIDsInt[i])
		}

		return updateIDs, nil
	} else {
		return api.GetUpdateIDs(sdc, owner.GetOwnerType(), termHmac) //todo get sorted doc updateID
	}
}

// remove owner's all search indexes(term + sortedDoc)
func RemoveSearchIndex(sdc client.StrongDocClient, owner SearchIdxOwner) error {
	if utils.TestLocal {
		return os.RemoveAll(GetSearchIdxPathPrefix(owner))
	} else {
		return api.RemoveSearchIndexes(sdc, owner.GetOwnerType())
	}
}
