package commands

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"strings"

	"github.com/dboxed/dboxed-common/db/migrator"
	config2 "github.com/dboxed/dboxed-volume/pkg/config"
	"github.com/dboxed/dboxed-volume/pkg/db/migration/postgres"
	"github.com/dboxed/dboxed-volume/pkg/db/migration/sqlite"
	"github.com/dboxed/dboxed-volume/pkg/server"
	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type ServerRunCmd struct {
	Config string `help:"Config file" type:"existingfile"`
}

func (cmd *ServerRunCmd) Run() error {
	ctx := context.Background()

	config, err := config2.LoadConfig(cmd.Config)
	if err != nil {
		return err
	}

	db, err := initDB(ctx, *config)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, "config", config)
	ctx = context.WithValue(ctx, "db", db)

	s, err := server.NewDboxedVolumeServer(ctx, *config)
	if err != nil {
		return err
	}

	err = s.InitGin()
	if err != nil {
		return err
	}

	err = s.InitHuma()
	if err != nil {
		return err
	}

	err = s.InitApi(ctx)
	if err != nil {
		return err
	}

	return s.ListenAndServe(ctx)
}

func openDB(ctx context.Context, config config2.Config, enableSqliteFKs bool) (*sqlx.DB, error) {
	purl, err := url.Parse(config.DB.Url)
	if err != nil {
		return nil, err
	}

	var sqlxDb *sqlx.DB
	if purl.Scheme == "sqlite3" {
		q := purl.Query()
		if enableSqliteFKs {
			if !q.Has("_foreign_keys") {
				q.Set("_foreign_keys", "on")
				purl.RawQuery = q.Encode()
			}
		}
		dbfile := strings.Replace(purl.String(), "sqlite3://", "", 1)

		sqlxDb, err = sqlx.Open("sqlite3", dbfile)
		if err != nil {
			return nil, err
		}
	} else if purl.Scheme == "postgresql" {
		sqlxDb, err = sqlx.Open("pgx", purl.String())
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unsupported db url: %s", config.DB.Url)
	}

	sqlxDb.SetMaxIdleConns(8)
	sqlxDb.SetMaxOpenConns(16)

	return sqlxDb, nil
}

func migrateDB(ctx context.Context, config config2.Config) error {
	db, err := openDB(ctx, config, false)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	err = migrator.Migrate(ctx, db, map[string]fs.FS{
		"pgx":     postgres.E,
		"sqlite3": sqlite.E,
	})
	if err != nil {
		return err
	}
	return nil
}

func initDB(ctx context.Context, config config2.Config) (*sqlx.DB, error) {
	slog.InfoContext(ctx, "initializing database")

	if config.DB.Migrate {
		slog.InfoContext(ctx, "migrating database")
		err := migrateDB(ctx, config)
		if err != nil {
			return nil, err
		}
	}

	db, err := openDB(ctx, config, true)
	if err != nil {
		return nil, err
	}

	return db, nil
}
