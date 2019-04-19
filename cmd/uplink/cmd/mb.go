// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"storj.io/storj/internal/fpath"
	"storj.io/storj/internal/memory"
	"storj.io/storj/lib/uplink"
	"storj.io/storj/pkg/process"
	"storj.io/storj/pkg/storj"
)

func init() {
	addCmd(&cobra.Command{
		Use:   "mb",
		Short: "Create a new bucket",
		RunE:  makeBucket,
	}, RootCmd)
}

func makeBucket(cmd *cobra.Command, args []string) error {
	ctx := process.Ctx(cmd)

	if len(args) == 0 {
		return fmt.Errorf("No bucket specified for creation")
	}

	dst, err := fpath.New(args[0])
	if err != nil {
		return err
	}

	if dst.IsLocal() {
		return fmt.Errorf("No bucket specified, use format sj://bucket/")
	}

	if dst.Path() != "" {
		return fmt.Errorf("Nested buckets not supported, use format sj://bucket/")
	}

	project, err := cfg.GetProject(ctx)
	if err != nil {
		return fmt.Errorf("Error setting up project: %+v\n", err)
	}
	defer func() {
		err = project.Close()
		if err != nil {
			fmt.Printf("Error closing project: %+v\n", err)
		}
	}()

	_, _, err = project.GetBucketInfo(ctx, dst.Bucket())
	if err == nil {
		return fmt.Errorf("Bucket already exists")
	}
	if !storj.ErrBucketNotFound.Has(err) {
		return err
	}

	bucketCfg := &uplink.BucketConfig{}
	//TODO (alex): make segment size customizable
	bucketCfg.Volatile = struct {
		RedundancyScheme storj.RedundancyScheme
		SegmentsSize     memory.Size
	}{
		RedundancyScheme: cfg.GetRedundancyScheme(),
	}

	_, err = project.CreateBucket(ctx, dst.Bucket(), bucketCfg)
	if err != nil {
		return err
	}

	fmt.Printf("Bucket %s created\n", dst.Bucket())

	return nil
}
