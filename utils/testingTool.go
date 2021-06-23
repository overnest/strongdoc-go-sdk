package utils

var booksDir string

func init() {
	dir, err := FetchFileLoc("./testDocuments/books")
	if err == nil {
		booksDir = dir
	}
}

func GetInitialTestDocumentDir() string {
	return booksDir
}
