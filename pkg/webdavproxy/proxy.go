package webdavproxy

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/dboxed/dboxed-volume/pkg/nats/dproto"
	"github.com/dboxed/dboxed-volume/pkg/nats/nats_handlers"
	"github.com/dboxed/dboxed-volume/pkg/s3proxy"
	"github.com/nats-io/nats.go"
	"golang.org/x/net/webdav"
)

type Proxy struct {
	nc             *nats.Conn
	repositoryUuid string

	listenAddr string
	httpServer *http.Server

	listCache map[string]*dproto.S3ProxyListObjectsReply

	cnt atomic.Int32
	m   sync.Mutex
}

func NewProxy(nc *nats.Conn, repositoryUuid string, listenAddr string) (*Proxy, error) {
	p := &Proxy{
		nc:             nc,
		repositoryUuid: repositoryUuid,
		listenAddr:     listenAddr,

		listCache: map[string]*dproto.S3ProxyListObjectsReply{},
	}
	return p, nil
}

func (p *Proxy) Start(ctx context.Context) error {
	listenAddr, err := net.ResolveTCPAddr("tcp", p.listenAddr)
	if err != nil {
		return err
	}

	l, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		return err
	}

	slog.Info("starting webdav proxy", slog.Any("listenAddr", l.Addr().String()))
	p.httpServer = &http.Server{
		Handler: p.handler(),
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}
	go func() {
		err := p.httpServer.Serve(l)
		if err != nil {
			slog.Error("webdav proxy server serve exited with error", slog.Any("error", err))
		}
	}()
	return nil
}

func (p *Proxy) handler() http.Handler {
	wh := &webdav.Handler{
		FileSystem: p,
		LockSystem: webdav.NewMemLS(),
	}

	wrapper := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == "HEAD" || request.Method == "GET" {
			// we're not interested in returning the content-type. the peeking that the webdav server does for this
			// slows down the whole process.
			writer.Header().Set("Content-Type", "")
		}
		wh.ServeHTTP(writer, request)
	})

	return wrapper
}

func (p *Proxy) natsListObjects(prefix string) (*dproto.S3ProxyListObjectsReply, error) {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	p.m.Lock()
	c, ok := p.listCache[prefix]
	p.m.Unlock()

	if ok {
		return c, nil
	}

	slog.Info("natsListObjects", slog.Any("prefix", prefix))
	c, err := nats_handlers.Request[*dproto.S3ProxyListObjectsReply](p.nc, s3proxy.NatsServiceName+".list-objects", &dproto.S3ProxyListObjectsRequest{
		RepositoryUuid: p.repositoryUuid,
		Prefix:         prefix,
	})
	if err != nil {
		return nil, err
	}

	p.m.Lock()
	p.listCache[prefix] = c
	p.m.Unlock()
	return c, nil
}

func (p *Proxy) parentPrefix(absName string) string {
	name := strings.TrimPrefix(absName, "/")
	name = strings.TrimSuffix(name, "/")
	parent := filepath.Dir(name)
	if parent == "." {
		parent = ""
	} else {
		parent = parent + "/"
	}
	return parent
}

func (p *Proxy) openReadFile(absName string) (*readFile, error) {
	f := &readFile{
		fileBase: fileBase{
			p:    p,
			name: absName,
		},
	}

	if absName == "/" {
		// root dir
		f.fi = &fileInfo{
			prefix: "",
		}
		return f, nil
	}

	parentList, err := p.natsListObjects(p.parentPrefix(absName))
	if err != nil {
		return nil, err
	}

	hasSlashSuffix := strings.HasSuffix(absName, "/")
	baseName := strings.TrimSuffix(absName, "/")
	baseName = path.Base(baseName)

	for _, cp := range parentList.CommonPrefixes {
		fi := &fileInfo{prefix: cp}
		if fi.Name() == baseName {
			f.fi = fi
			return f, nil
		}
	}
	for _, oi := range parentList.Objects {
		fi := &fileInfo{oi: oi}

		if fi.Name() == baseName {
			f.fi = fi

			if hasSlashSuffix {
				return nil, fmt.Errorf("not a directory")
			}

			return f, nil
		}
	}

	return nil, os.ErrNotExist
}

func (p *Proxy) createFile(name string) (*writeFile, error) {
	trimmedName := strings.TrimPrefix(name, "/")
	f := &writeFile{
		fileBase: fileBase{
			p:    p,
			name: name,
			fi: &fileInfo{
				oi: &dproto.S3ObjectInfo{
					Key: trimmedName,
				},
			},
		},
	}
	err := f.Start()
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (p *Proxy) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return nil
}

func (p *Proxy) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	isPut := flag == os.O_RDWR|os.O_CREATE|os.O_TRUNC
	isGet := flag == os.O_RDONLY

	if isGet {
		f, err := p.openReadFile(name)
		if err != nil {
			return nil, err
		}
		return f, nil
	} else if isPut {
		f, err := p.createFile(name)
		if err != nil {
			return nil, err
		}
		return f, nil
	} else {
		return nil, fmt.Errorf("unsupporrted OpenFile invocation: 0x%x", flag)
	}
}

func (p *Proxy) forgetCache(prefix string) {
	p.m.Lock()
	defer p.m.Unlock()

	for {
		_, ok := p.listCache[prefix]
		if ok {
			delete(p.listCache, prefix)
			break
		} else {
			if prefix == "/" {
				return
			}
			prefix = path.Dir(strings.TrimSuffix(prefix, "/"))
			if prefix == "." {
				prefix = ""
			}
			prefix += "/"
		}
	}
}

func (p *Proxy) remove(ctx context.Context, name string) error {
	slog.Info("remove", slog.Any("name", name))
	_, err := nats_handlers.Request[*dproto.S3ProxyRemoveObjectReply](p.nc, s3proxy.NatsServiceName+".remove-object", &dproto.S3ProxyRemoveObjectRequest{
		RepositoryUuid: p.repositoryUuid,
		ObjectName:     name,
	})
	if err != nil {
		return err
	}
	return nil
}

func (p *Proxy) listRecursive(ctx context.Context, name string, ret *[]string) error {
	f, err := p.openReadFile(name)
	if err != nil {
		return err
	}
	defer f.Close()

	if !f.fi.IsDir() {
		*ret = append(*ret, name)
		return nil
	} else {
		dis, err := f.Readdir(0)
		if err != nil {
			return err
		}
		for _, di := range dis {
			err = p.listRecursive(ctx, path.Join(name, di.Name()), ret)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Proxy) RemoveAll(ctx context.Context, name string) error {
	var files []string
	err := p.listRecursive(ctx, name, &files)
	if err != nil {
		return err
	}

	for _, f := range files {
		err = p.remove(ctx, f)
		p.forgetCache(p.parentPrefix(f))
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Proxy) Rename(ctx context.Context, oldName, newName string) error {
	f, err := p.openReadFile(oldName)
	if err != nil {
		return err
	}
	defer f.Close()

	if f.fi.IsDir() {
		return fmt.Errorf("renaming dirs not supported")
	}

	trimmedOldName := strings.TrimPrefix(oldName, "/")
	trimmedNewName := strings.TrimPrefix(newName, "/")

	slog.Info("rename", slog.Any("oldName", oldName), slog.Any("newName", newName))
	_, err = nats_handlers.Request[*dproto.S3ProxyRenameObjectReply](p.nc, s3proxy.NatsServiceName+".rename-object", &dproto.S3ProxyRenameObjectRequest{
		RepositoryUuid: p.repositoryUuid,
		OldObjectName:  trimmedOldName,
		NewObjectName:  trimmedNewName,
	})
	p.forgetCache(p.parentPrefix(oldName))
	p.forgetCache(p.parentPrefix(newName))
	if err != nil {
		return err
	}
	return nil
}

func (p *Proxy) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	f, err := p.openReadFile(name)
	if err != nil {
		return nil, err
	}
	return f.Stat()
}
