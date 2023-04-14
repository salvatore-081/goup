package internal

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
	"github.com/salvatore-081/goup/internal/middlewares"
)

type Resolver struct {
	Host         string
	XAPIKey      string
	BadgerDB     *badger.DB
	DockerClient *client.Client
}

func (r *Resolver) Create(xAPIKey string) (e error) {
	r.XAPIKey = xAPIKey

	r.BadgerDB, e = badger.Open(badger.DefaultOptions("./data").WithLogger(middlewares.BadgerLogger{}))
	if e != nil {
		return e
	}

	e = r.BadgerDB.Update(func(txn *badger.Txn) error {
		_, e := txn.Get([]byte("volumes"))
		if e != nil {
			switch {
			case e == badger.ErrKeyNotFound:
				b, e := json.Marshal([]string{})
				if e != nil {
					return e
				}
				e = txn.SetEntry(badger.NewEntry([]byte("volumes"), b))
				if e != nil {
					return e
				}
			default:
				return e
			}
		}
		return nil
	})
	if e != nil {
		return e
	}

	r.DockerClient, e = client.NewClientWithOpts()
	if e != nil {
		return e
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			e := r.BadgerDB.RunValueLogGC(0.5)
			if e != nil {
				log.Debug().Err(e).Str("SERVICE", "DB").Msg("")
			}
		}
	}()

	return nil
}

func (r *Resolver) Close(ctx context.Context) error {
	e := make(chan error, 1)

	go func() {
		e <- r.BadgerDB.Close()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-e:
		return e
	}
}
