// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"storj.io/storj/internal/fpath"
	libuplink "storj.io/storj/lib/uplink"
	"storj.io/storj/pkg/process"
	"storj.io/storj/pkg/storj"
)

func init() {
	addCmd(&cobra.Command{
		Use:   "rb",
		Short: "Remove an empty bucket",
		RunE:  deleteBucket,
	}, RootCmd)
}

func deleteBucket(cmd *cobra.Command, args []string) error {
	ctx := process.Ctx(cmd)

	if len(args) == 0 {
		return fmt.Errorf("No bucket specified for deletion")
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
		return err
	}
	defer func() {
		err := project.Close()
		if err != nil {
			fmt.Printf("error closing project: %+v\n", err)
		}
	}()

	var access libuplink.EncryptionAccess
	copy(access.Key[:], []byte(cfg.Enc.Key))

	bucket, err := project.OpenBucket(ctx, dst.Bucket(), &access)
	if err != nil {
		return convertError(err, dst)
	}

	list, err := bucket.ListObjects(ctx, &storj.ListOptions{Direction: storj.After, Recursive: true, Limit: 1})
	if err != nil {
		return convertError(err, dst)
	}

	if len(list.Items) > 0 {
		return fmt.Errorf("Bucket not empty: %s", dst.Bucket())
	}

	err = project.DeleteBucket(ctx, dst.Bucket())
	if err != nil {
		return convertError(err, dst)
	}

	fmt.Printf("Bucket %s deleted\n", dst.Bucket())

	return nil
}
