package webdavproxy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dboxed/dboxed-volume/pkg/nats/dproto"
	"golang.org/x/net/webdav"
)

type fileInfo struct {
	prefix string
	oi     *dproto.S3ObjectInfo
}

func (f *fileInfo) Name() string {
	if f.oi != nil {
		return filepath.Base(f.oi.Key)
	} else {
		return filepath.Base(strings.TrimSuffix(f.prefix, "/"))
	}
}
func (f *fileInfo) Size() int64 {
	if f.oi != nil {
		return f.oi.Size
	} else {
		return 0
	}
}
func (f *fileInfo) Mode() os.FileMode {
	if f.oi != nil {
		return 0
	} else {
		return os.ModeDir
	}
}

func (f *fileInfo) ModTime() time.Time {
	if f.oi != nil {
		return f.oi.LastModified.AsTime()
	} else {
		return time.Time{}
	}
}
func (f *fileInfo) IsDir() bool      { return f.Mode().IsDir() }
func (f *fileInfo) Sys() interface{} { return nil }

func (f *fileInfo) ContentType(ctx context.Context) (string, error) {
	if f.oi != nil {
		return "", nil
	} else {
		return "", nil
	}
}

func (f *fileInfo) ETag(ctx context.Context) (string, error) {
	if f.oi != nil {
		return f.oi.Etag, nil
	}
	return "", webdav.ErrNotImplemented
}
