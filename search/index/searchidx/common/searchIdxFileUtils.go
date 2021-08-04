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

func NewUpdateIDV1() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

//////////////////////////////////////////////////////////////////
//
//                     Local Testing Path
//
//////////////////////////////////////////////////////////////////

// getSearchIdxPathPrefix gets the search index path prefix
// return /tmp/search/<owner>/sidx
func getSearchIdxPathPrefix(owner SearchIdxOwner) string {
	return fmt.Sprintf("/tmp/search/%v/sidx", owner)
}

// getSearchIdxPath gets the base path of the search index
// return /tmp/search/<owner>/sidx/<termID>/updateID
func getSearchIdxPath(owner SearchIdxOwner, termID, updateID string) string {
	return fmt.Sprintf("%v/%v/%v", getSearchIdxPathPrefix(owner), termID, updateID)
}

//  return /tmp/search/<owner>/sidx/<termID>/updateID/searchterm
func getSearchTermIdxPath(owner SearchIdxOwner, termID, updateID string) string {
	return fmt.Sprintf("%v/searchterm", getSearchIdxPath(owner, termID, updateID))
}

//  return /tmp/search/<owner>/sidx/<termID>/updateID/sortdoc
func getSearchSortDocIdxPath(owner SearchIdxOwner, termID, updateID string) string {
	return fmt.Sprintf("%v/sortdoc", getSearchIdxPath(owner, termID, updateID))
}

func OpenSearchTermIndexReader(sdc client.StrongDocClient, owner SearchIdxOwner, termID string, updateID string) (io.ReadCloser, uint64, error) {
	if utils.TestLocal {
		path := getSearchTermIdxPath(owner, termID, updateID)
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
	reader, err := api.NewSearchTermIdxReader(sdc, owner.GetOwnerType(), termID, updateID)
	if err != nil {
		return nil, 0, err
	}

	size, err := api.GetSearchTermIndexSize(sdc, owner.GetOwnerType(), termID, updateID)
	if err != nil {
		return nil, 0, err
	}
	return reader, size, nil
}

func OpenSearchSortDocIndexReader(sdc client.StrongDocClient, owner SearchIdxOwner, termID string, updateID string) (io.ReadCloser, uint64, error) {
	if utils.TestLocal {
		path := getSearchSortDocIdxPath(owner, termID, updateID)
		//fmt.Println("ssdi reader path", path)
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
	reader, err := api.NewSearchSortedDocIdxReader(sdc, owner.GetOwnerType(), termID, updateID)
	if err != nil {
		return nil, 0, err
	}

	size, err := api.GetSearchSortedDocIdxSize(sdc, owner.GetOwnerType(), termID, updateID)
	if err != nil {
		return nil, 0, err
	}
	return reader, size, nil
}

func OpenSearchSortDocIndexWriter(sdc client.StrongDocClient, owner SearchIdxOwner, termHmac string, updateID string) (io.WriteCloser, error) {
	if utils.TestLocal {
		path := getSearchSortDocIdxPath(owner, termHmac, updateID)
		//fmt.Println("ssdi writer path", path)

		return utils.MakeDirAndCreateFile(path)
	}
	return api.NewSearchSortedDocIdxWriter(sdc, owner.GetOwnerType(), termHmac, updateID)
}

func OpenSearchTermIndexWriter(sdc client.StrongDocClient, owner SearchIdxOwner, termHmac string) (writer io.WriteCloser, updateID string, err error) {
	if utils.TestLocal {
		updateID = NewUpdateIDV1()
		path := getSearchTermIdxPath(owner, termHmac, updateID)
		writer, err = utils.MakeDirAndCreateFile(path)
		return
	}
	return api.NewSearchTermIdxWriter(sdc, owner.GetOwnerType(), termHmac)
}

//////////////////////////////////////////////////////////////////
//
//                         Update ID
//
//////////////////////////////////////////////////////////////////

func GetUpdateIDs(sdc client.StrongDocClient, owner SearchIdxOwner, termID string) ([]string, error) {
	if utils.TestLocal {
		path := getSearchIdxPath(owner, termID, "")

		files, err := ioutil.ReadDir(path)
		if err != nil {
			if _, ok := err.(*os.PathError); ok { // The term does not exist
				return nil, os.ErrNotExist
			}
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
		return api.GetUpdateIDs(sdc, owner.GetOwnerType(), termID) //todo get sorted doc updateID
	}
}

func GetLatestUpdateID(sdc client.StrongDocClient, owner SearchIdxOwner, termID string) (string, error) {
	updateIDs, err := GetUpdateIDs(sdc, owner, termID)
	if err != nil {
		return "", err
	}

	if updateIDs != nil {
		return updateIDs[0], nil
	}

	return "", nil
}

// remove owner's all search indexes(term + sortedDoc)
func RemoveSearchIndex(sdc client.StrongDocClient, owner SearchIdxOwner) error {
	if utils.TestLocal {
		return os.RemoveAll(getSearchIdxPathPrefix(owner))
	} else {
		return api.RemoveSearchIndexes(sdc, owner.GetOwnerType())
	}
}
