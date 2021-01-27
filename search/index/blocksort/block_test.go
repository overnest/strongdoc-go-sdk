package blocksort

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/utils"
	"gotest.tools/assert"
)

func TestTermIdxBlockV1(t *testing.T) {
	sourceFileName := "./testDocuments/enwik8.txt.gz"
	sourceFilePath, err := utils.FetchFileLoc(sourceFileName)

	outputFileName := "/tmp/blockSortOutput.txt"
	outfile, err := os.Create(outputFileName)
	assert.NilError(t, err)
	defer outfile.Close()

	var prevHighTerm string = ""
	var prevHighTermCount uint32 = 0

	total := 0

	for true {
		dtib := CreateDockTermIdxBlkV1(prevHighTerm, prevHighTermCount)

		tokenizer, err := utils.OpenFileTokenizer(sourceFilePath)
		assert.NilError(t, err)
		defer tokenizer.Close()

		for token, pos, err := tokenizer.NextToken(); err != io.EOF; token, pos, err = tokenizer.NextToken() {
			dtib.AddTerm(token)
			_ = pos
		}

		s, err := dtib.Serialize()
		assert.NilError(t, err)

		lowtermcount, _ := dtib.GetLowTermCount()
		hightermcount, _ := dtib.GetHighTermCount()
		fmt.Println(dtib.lowTerm, lowtermcount, dtib.highTerm, hightermcount, len(dtib.Terms), dtib.IsFull())
		fmt.Println(len(s), dtib.predictedJSONSize, DTI_BLOCK_SIZE_MAX)

		total += len(dtib.Terms)

		for _, t := range dtib.Terms {
			outfile.WriteString(t)
			outfile.WriteString("\n")
		}

		if !dtib.IsFull() {
			break
		}

		prevHighTerm = dtib.highTerm
		prevHighTermCount, err = dtib.GetHighTermCount()
		assert.NilError(t, err)

	}

	fmt.Println("total", total)

}
