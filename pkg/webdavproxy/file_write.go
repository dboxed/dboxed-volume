package webdavproxy

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"sync"

	"github.com/dboxed/dboxed-volume/pkg/nats/dproto"
	"github.com/dboxed/dboxed-volume/pkg/nats/nats_handlers"
	"github.com/dboxed/dboxed-volume/pkg/s3proxy"
	"golang.org/x/net/webdav"
)

type writeFile struct {
	fileBase

	presignedPut *dproto.S3ProxyPresignPutReply

	m            sync.Mutex
	uploadReq    *http.Request
	uploadResp   *http.Response
	uploadWriter *io.PipeWriter
	uploadErr    error
	uploadDone   chan struct{}
}

func (f *writeFile) presignPutUrl() error {
	slog.Info("presignPutUrl", slog.Any("name", f.name))
	rep, err := nats_handlers.Request[*dproto.S3ProxyPresignPutReply](f.p.nc, s3proxy.NatsServiceName+".presign-put", &dproto.S3ProxyPresignPutRequest{
		RepositoryUuid: f.p.repositoryUuid,
		ObjectName:     f.fi.oi.Key,
	})
	if err != nil {
		return err
	}
	f.presignedPut = rep
	return nil
}

func (f *writeFile) beginUpload(url string) error {
	bodyReader, bodyWriter := io.Pipe()

	f.m.Lock()
	defer f.m.Unlock()

	if f.uploadReq != nil {
		return fmt.Errorf("upload already started")
	}

	slog.Info("beginUpload", slog.Any("name", f.fi.Name()))

	req, err := http.NewRequest("PUT", url, bufio.NewReaderSize(bodyReader, 1024*64))
	if err != nil {
		return err
	}

	f.uploadReq = req
	f.uploadDone = make(chan struct{})
	f.uploadWriter = bodyWriter
	go func() {
		resp, err := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		f.handleUploadDone(resp, err)
	}()

	return nil
}

func (f *writeFile) handleUploadDone(resp *http.Response, err error) {
	f.m.Lock()
	defer f.m.Unlock()

	slog.Info("handleUploadDone", slog.Any("name", f.fi.Name()))

	f.uploadResp = resp
	f.uploadErr = err
	close(f.uploadDone)
}

func (f *writeFile) checkUploadErr() error {
	f.m.Lock()
	defer f.m.Unlock()
	if f.uploadErr != nil {
		return f.uploadErr
	}
	if f.uploadResp == nil {
		return nil
	}
	if f.uploadResp.StatusCode < 200 || f.uploadResp.StatusCode >= 300 {
		return fmt.Errorf("upload status: %s", f.uploadResp.Status)
	}
	return nil
}

func (f *writeFile) Start() error {
	err := f.presignPutUrl()
	if err != nil {
		return err
	}

	err = f.beginUpload(f.presignedPut.Url)
	if err != nil {
		return err
	}
	return nil
}

func (f *writeFile) Close() error {
	err := f.checkUploadErr()
	if err != nil {
		return err
	}

	slog.Info("Close", slog.Any("name", f.fi.Name()))

	err = f.uploadWriter.Close()
	if err != nil {
		f.m.Lock()
		if f.uploadErr != nil {
			f.uploadErr = err
		}
		f.m.Unlock()
		return err
	}

	slog.Info("wait for close", slog.Any("name", f.fi.Name()))
	<-f.uploadDone
	slog.Info("close done", slog.Any("name", f.fi.Name()))

	err = f.checkUploadErr()
	if err != nil {
		return err
	}

	f.p.forgetCache(f.p.parentPrefix(f.name))

	return nil
}

func (f *writeFile) Write(p []byte) (int, error) {
	//slog.Info("Write", slog.Any("name", f.fi.Name()), slog.Any("len", len(p)))
	return f.uploadWriter.Write(p)
}

func (f *writeFile) Seek(offset int64, whence int) (int64, error) {
	return 0, webdav.ErrNotImplemented
}

func (f *writeFile) Read(p []byte) (n int, err error) {
	return 0, webdav.ErrNotImplemented
}

func (f *writeFile) Readdir(count int) ([]fs.FileInfo, error) {
	return nil, webdav.ErrNotImplemented
}
