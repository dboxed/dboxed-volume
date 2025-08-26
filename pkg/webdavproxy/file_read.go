package webdavproxy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"golang.org/x/net/webdav"
)

const minChunkSize = 1
const maxChunkSize = 1024 * 1024 * 4
const speedUpAfterMinChunks = 4

type readFile struct {
	fileBase

	streamedChunks int
	buf            bytes.Buffer

	readOffset int64
}

func (f *readFile) fillBuf(offset int64, count int64) error {
	rangeHeader := fmt.Sprintf("bytes=%d-%d", offset, offset+count-1)

	slog.Info("GET chunks", slog.Any("name", f.name), slog.Any("offset", offset), slog.Any("count", count))

	req, err := http.NewRequest("GET", f.fi.oi.PresignedGetUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", rangeHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == 404 {
			return os.ErrNotExist
		}
		return fmt.Errorf("http status: %s", resp.Status)
	}

	f.buf.Reset()
	_, err = io.Copy(&f.buf, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func (f *readFile) Close() error {
	f.buf.Reset()
	return nil
}

func (f *readFile) Read(p []byte) (int, error) {
	if f.fi.IsDir() {
		return 0, fmt.Errorf("not a file")
	}

	if len(p) == 0 {
		return 0, nil
	}

	if f.buf.Len() == 0 {
		if f.readOffset >= f.fi.Size() {
			return 0, io.EOF
		}

		f.streamedChunks++

		chunkSize := minChunkSize
		if len(p) >= minChunkSize {
			chunkSize = len(p)
		}

		if f.streamedChunks >= speedUpAfterMinChunks && maxChunkSize > len(p) {
			chunkSize = maxChunkSize
		}

		err := f.fillBuf(f.readOffset, int64(chunkSize))
		if err != nil {
			return 0, err
		}
	}

	n, err := f.buf.Read(p)
	f.readOffset += int64(n)
	return n, err
}

func (f *readFile) Seek(offset int64, whence int) (int64, error) {
	oldOffset := f.readOffset
	defer func() {
		if f.readOffset != oldOffset && f.streamedChunks != 0 {
			f.streamedChunks = 0
		}
	}()

	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = f.readOffset + offset
	case io.SeekEnd:
		abs = f.fi.Size() + offset
	default:
		return 0, errors.New("invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("negative position")
	}

	delta := abs - f.readOffset
	if delta < 0 {
		f.buf.Reset()
	} else {
		f.buf.Next(int(delta))
	}

	f.readOffset = abs
	return abs, nil
}

func (f *readFile) Readdir(count int) ([]fs.FileInfo, error) {
	if !f.fi.IsDir() {
		return nil, fmt.Errorf("not a directory")
	}

	rep, err := f.p.natsListObjects(f.fi.prefix)
	if err != nil {
		return nil, err
	}

	var ret []fs.FileInfo
	for _, cp := range rep.CommonPrefixes {
		ret = append(ret, &fileInfo{
			prefix: cp,
		})
	}
	for _, x := range rep.Objects {
		ret = append(ret, &fileInfo{
			oi: x,
		})
	}

	if count != 0 && len(ret) > count {
		ret = ret[:count]
	}

	return ret, nil
}

func (f *readFile) Write(p []byte) (n int, err error) {
	return 0, webdav.ErrNotImplemented
}
