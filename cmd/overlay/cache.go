// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"storj.io/storj/pkg/overlay"
	"storj.io/storj/satellite/satellitedb"
)

type cacheConfig struct {
	NodesPath string `help:"the path to a JSON file containing an object with IP keys and nodeID values"`
	Database  string `help:"overlay database connection string" default:"sqlite3://$CONFDIR/master.db"`
}

func (c cacheConfig) open(ctx context.Context) (cache *overlay.Cache, dbClose func(), err error) {
	database, err := satellitedb.New(zap.L().Named("db"), c.Database)
	if err != nil {
		return nil, nil, errs.New("error connecting to database: %+v", err)
	}
	dbClose = func() {
		err := database.Close()
		if err != nil {
			fmt.Printf("error closing connection to database: %+v\n", err)
		}
	}

	return overlay.NewCache(zap.L(), database.OverlayCache(), overlay.NodeSelectionConfig{OnlineWindow: time.Hour}), dbClose, nil
}
