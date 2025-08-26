package flags

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

type GlobalFlags struct {
	Debug bool `help:"Enable debugging mode"`

	NatsUrl  string `help:"Specify the nats url"`
	NatsNkey string `help:"Specify the nkey file" type:"existingfile"`
}

func (f *GlobalFlags) ConnectNats() (*nats.Conn, error) {
	if f.NatsNkey == "" {
		return nil, fmt.Errorf("missing nats nkey")
	}

	nkeyOpt, err := nats.NkeyOptionFromSeed(f.NatsNkey)
	if err != nil {
		return nil, err
	}
	nc, err := nats.Connect(f.NatsUrl, nkeyOpt)
	if err != nil {
		return nil, err
	}
	return nc, nil
}
