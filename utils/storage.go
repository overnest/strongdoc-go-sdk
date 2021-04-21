package utils

import (
	"bufio"
	"bytes"
	"io"
	"unicode"
)

//////////////////////////////////////////////////////////////////
//
//               storage for tokenizer
//
//////////////////////////////////////////////////////////////////

type Source interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
}

type Storage interface {
	nextRawToken() ([]byte, rune, error)
	close() error
	reset() error
}

type storage struct {
	fileType FileType
	source   Source
	bReader  *bufio.Reader
	closer   io.Closer
}

func openStorage(source Source) (Storage, error) {
	fileType, reader, closer, err := getFileTypeAndReaderCloser(source)
	if err != nil {
		return nil, err
	}
	bReader := bufio.NewReader(reader)
	return &storage{
		fileType: fileType,
		source:   source,
		bReader:  bReader,
		closer:   closer,
	}, nil
}

// read non-space bytes
// return data []byte, endingChar rune, err error
// data == nil, r = unicode.ReplacementChar err = io.EOF, reach end of file
// data == nil, r = unicode.ReplacementChar, err != nil, some other error
// data != nil, r = space character, err = nil, read some data
func (storage *storage) nextRawToken() ([]byte, rune, error) {
	bReader := storage.bReader
	var err error
	var r = unicode.ReplacementChar
	var buffer bytes.Buffer
	bufWriter := bufio.NewWriter(&buffer)

	for {
		r, _, err = bReader.ReadRune()
		// reach end or space character
		if err == io.EOF || unicode.IsSpace(r) {
			break
		}
		if err != nil {
			return nil, r, err
		}

		_, err = bufWriter.WriteRune(r)
		if err != nil {
			return nil, r, err
		}

	}

	bufWriter.Flush()
	if buffer.Len() > 0 && err != nil {
		err = nil
	}

	return buffer.Bytes(), r, err
}

// Close closes the storage. It does not close the underlying source
func (storage *storage) close() error {
	if storage.closer != nil {
		return storage.closer.Close()
	}
	return nil
}

func (storage *storage) reset() error {
	err := storage.close()
	if err != nil {
		return err
	}
	reader, closer, err := resetFile(storage.fileType, storage.source)
	if err != nil {
		return err
	}
	bReader := bufio.NewReader(reader)
	storage.bReader = bReader
	storage.closer = closer
	return nil
}
