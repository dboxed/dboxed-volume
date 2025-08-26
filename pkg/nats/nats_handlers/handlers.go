package nats_handlers

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
)

func BuildHandler[PReq proto.Message, PRep proto.Message](ctx context.Context, sem *semaphore.Weighted, handler func(ctx context.Context, r PReq) (PRep, error)) micro.HandlerFunc {
	return func(req micro.Request) {
		t := reflect.TypeFor[PReq]().Elem()
		preq := reflect.New(t).Interface().(proto.Message)

		err := proto.Unmarshal(req.Data(), preq)
		if err != nil {
			_ = req.Error("unmarshal", err.Error(), nil)
			return
		}

		go func() {
			err := sem.Acquire(ctx, 1)
			if err != nil {
				return
			}
			defer sem.Release(1)

			prep, err := handler(ctx, preq.(PReq))
			if err != nil {
				_ = req.Error("handling", err.Error(), nil)
				return
			}
			b, err := proto.Marshal(prep)
			if err != nil {
				_ = req.Error("marshal", err.Error(), nil)
				return
			}
			err = req.Respond(b)
			if err != nil {
				slog.Error("respond failed", slog.Any("subject", req.Subject()))
			}
		}()
	}
}

func Request[PRep proto.Message, PReq proto.Message](nc *nats.Conn, sub string, preq PReq) (PRep, error) {
	var z PRep

	b, err := proto.Marshal(preq)
	if err != nil {
		return z, err
	}
	rep, err := nc.Request(sub, b, time.Second*30)
	if err != nil {
		return z, err
	}
	herr := rep.Header.Get(micro.ErrorHeader)
	if herr != "" {
		errCode := rep.Header.Get(micro.ErrorCodeHeader)
		return z, fmt.Errorf("got error from service. code=%s, description=%s", errCode, herr)
	}

	t := reflect.TypeFor[PRep]().Elem()
	prep := reflect.New(t).Interface().(proto.Message)
	err = proto.Unmarshal(rep.Data, prep)
	if err != nil {
		return z, err
	}
	return prep.(PRep), nil
}
