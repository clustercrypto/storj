// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package overlay_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zeebo/errs"

	"storj.io/storj/internal/testcontext"
	"storj.io/storj/internal/testplanet"
	"storj.io/storj/pkg/overlay"
	"storj.io/storj/pkg/storj"
)

func TestNodeSelection(t *testing.T) {
	t.Skip("flaky")
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 10, UplinkCount: 1,
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		var err error
		satellite := planet.Satellites[0]

		// This sets a reputable audit count for a certain number of nodes.
		for i, node := range planet.StorageNodes {
			for k := 0; k < i; k++ {
				_, err := satellite.DB.OverlayCache().UpdateStats(ctx, &overlay.UpdateRequest{
					NodeID:       node.ID(),
					IsUp:         true,
					AuditSuccess: true,
				})
				assert.NoError(t, err)
			}
		}

		// ensure all storagenodes are in overlay service
		for _, storageNode := range planet.StorageNodes {
			err = satellite.Overlay.Service.Put(ctx, storageNode.ID(), storageNode.Local().Node)
			assert.NoError(t, err)
		}

		type test struct {
			Preferences    overlay.NodeSelectionConfig
			ExcludeCount   int
			RequestCount   int
			ExpectedCount  int
			ShouldFailWith *errs.Class
		}

		for i, tt := range []test{
			{ // all reputable nodes, only reputable nodes requested
				Preferences: overlay.NodeSelectionConfig{
					AuditCount:        0,
					NewNodePercentage: 0,
					OnlineWindow:      time.Hour,
				},
				RequestCount:  5,
				ExpectedCount: 5,
			},
			{ // all reputable nodes, reputable and new nodes requested
				Preferences: overlay.NodeSelectionConfig{
					AuditCount:        0,
					NewNodePercentage: 1,
					OnlineWindow:      time.Hour,
				},
				RequestCount:  5,
				ExpectedCount: 5,
			},
			{ // all reputable nodes except one, reputable and new nodes requested
				Preferences: overlay.NodeSelectionConfig{
					AuditCount:        1,
					NewNodePercentage: 1,
					OnlineWindow:      time.Hour,
				},
				RequestCount:  5,
				ExpectedCount: 6,
			},
			{ // 50-50 reputable and new nodes, reputable and new nodes requested (new node ratio 1.0)
				Preferences: overlay.NodeSelectionConfig{
					AuditCount:        5,
					NewNodePercentage: 1,
					OnlineWindow:      time.Hour,
				},
				RequestCount:  2,
				ExpectedCount: 4,
			},
			{ // 50-50 reputable and new nodes, reputable and new nodes requested (new node ratio 0.5)
				Preferences: overlay.NodeSelectionConfig{
					AuditCount:        5,
					NewNodePercentage: 0.5,
					OnlineWindow:      time.Hour,
				},
				RequestCount:  4,
				ExpectedCount: 6,
			},
			{ // all new nodes except one, reputable and new nodes requested (happy path)
				Preferences: overlay.NodeSelectionConfig{
					AuditCount:        8,
					NewNodePercentage: 1,
					OnlineWindow:      time.Hour,
				},
				RequestCount:  1,
				ExpectedCount: 2,
			},
			{ // all new nodes except one, reputable and new nodes requested (not happy path)
				Preferences: overlay.NodeSelectionConfig{
					AuditCount:        9,
					NewNodePercentage: 1,
					OnlineWindow:      time.Hour,
				},
				RequestCount:   2,
				ExpectedCount:  3,
				ShouldFailWith: &overlay.ErrNotEnoughNodes,
			},
			{ // all new nodes, reputable and new nodes requested
				Preferences: overlay.NodeSelectionConfig{
					AuditCount:        50,
					NewNodePercentage: 1,
					OnlineWindow:      time.Hour,
				},
				RequestCount:   2,
				ExpectedCount:  2,
				ShouldFailWith: &overlay.ErrNotEnoughNodes,
			},
			{ // audit threshold edge case (1)
				Preferences: overlay.NodeSelectionConfig{
					AuditCount:        9,
					NewNodePercentage: 0,
					OnlineWindow:      time.Hour,
				},
				RequestCount:  1,
				ExpectedCount: 1,
			},
			{ // audit threshold edge case (2)
				Preferences: overlay.NodeSelectionConfig{
					AuditCount:        0,
					NewNodePercentage: 1,
					OnlineWindow:      time.Hour,
				},
				RequestCount:  1,
				ExpectedCount: 1,
			},
			{ // excluded node ids being excluded
				Preferences: overlay.NodeSelectionConfig{
					AuditCount:        5,
					NewNodePercentage: 0,
					OnlineWindow:      time.Hour,
				},
				ExcludeCount:   7,
				RequestCount:   5,
				ExpectedCount:  3,
				ShouldFailWith: &overlay.ErrNotEnoughNodes,
			},
		} {
			t.Logf("#%2d. %+v", i, tt)
			service := planet.Satellites[0].Overlay.Service

			var excludedNodes []storj.NodeID
			for _, storageNode := range planet.StorageNodes[:tt.ExcludeCount] {
				excludedNodes = append(excludedNodes, storageNode.ID())
			}

			response, err := service.FindStorageNodesWithPreferences(ctx, overlay.FindStorageNodesRequest{
				FreeBandwidth:  0,
				FreeDisk:       0,
				RequestedCount: tt.RequestCount,
				ExcludedNodes:  excludedNodes,
			}, &tt.Preferences)

			t.Log(len(response), err)
			if tt.ShouldFailWith != nil {
				assert.Error(t, err)
				assert.True(t, tt.ShouldFailWith.Has(err))
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.ExpectedCount, len(response))
		}
	})
}
