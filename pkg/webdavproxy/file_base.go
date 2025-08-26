package webdavproxy

import (
	"io/fs"
)

type fileBase struct {
	p    *Proxy
	name string

	fi *fileInfo
}

func (f *fileBase) Stat() (fs.FileInfo, error) {
	return f.fi, nil
}
