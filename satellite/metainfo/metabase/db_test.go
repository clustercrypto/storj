// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package metabase_test

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"storj.io/common/testcontext"
	_ "storj.io/storj/private/dbutil/cockroachutil" // register cockroach driver
	"storj.io/storj/satellite/metainfo/metabase"
	"storj.io/storj/satellite/satellitedb/satellitedbtest"
)

var databases = flag.String("databases", os.Getenv("STORJ_TEST_DATABASES"), "databases to use for testing")

func All(t *testing.T, fn func(ctx *testcontext.Context, t *testing.T, db *metabase.DB)) {
	type dbinfo struct {
		name    string
		driver  string
		connstr string
	}

	infos := []dbinfo{
		{"pg", "pgx", "postgres://storj:storj-pass@localhost/metabase?sslmode=disable"},
		{"crdb", "pgx", "cockroach://root@localhost:26257/metabase?sslmode=disable"},
	}
	if *databases != "" {
		infos = nil
		for _, db := range strings.Split(*databases, ";") {
			toks := strings.Split(strings.TrimSpace(db), "|")
			infos = append(infos, dbinfo{toks[0], toks[1], toks[2]})
		}
	}

	for _, info := range infos {
		info := info
		t.Run(info.name, func(t *testing.T) {
			t.Parallel()

			ctx := testcontext.New(t)
			defer ctx.Cleanup()

			db, err := satellitedbtest.CreateMetabaseDB(ctx, zaptest.NewLogger(t), t.Name(), "M", 0, satellitedbtest.Database{
				Name:    info.name,
				URL:     info.connstr,
				Message: "",
			})
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := db.Close(); err != nil {
					t.Error(err)
				}
			}()

			if err := db.MigrateToLatest(ctx); err != nil {
				t.Fatal(err)
			}

			fn(ctx, t, db.InternalImplementation().(*metabase.DB))
		})
	}
}

func TestSetup(t *testing.T) {
	All(t, func(ctx *testcontext.Context, t *testing.T, db *metabase.DB) {
		err := db.Ping(ctx)
		require.NoError(t, err)

		_, err = db.TestingGetState(ctx)
		require.NoError(t, err)
	})
}
