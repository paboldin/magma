/*
 Copyright (c) Facebook, Inc. and its affiliates.
 All rights reserved.

 This source code is licensed under the BSD-style license found in the
 LICENSE file in the root directory of this source tree.
*/

package servicers

import (
	"context"

	"magma/orc8r/cloud/go/services/state/indexer/reindex"
	indexer_protos "magma/orc8r/cloud/go/services/state/protos"
	"magma/orc8r/lib/go/protos"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type indexerServicer struct {
	reindexer   reindex.Reindexer
	autoEnabled bool
}

func NewIndexerServicer(reindexer reindex.Reindexer, autoReindexEnabled bool) indexer_protos.IndexerServer {
	return &indexerServicer{reindexer: reindexer, autoEnabled: autoReindexEnabled}
}

func (srv *indexerServicer) GetIndexers(ctx context.Context, req *indexer_protos.GetIndexersRequest) (*indexer_protos.GetIndexersResponse, error) {
	if err := validateCtx(ctx); err != nil {
		return nil, err
	}

	versions, err := srv.reindexer.GetIndexerVersions()
	if err != nil {
		return nil, internalErr(err, "error getting indexer versions from reindex job queue")
	}

	ret := &indexer_protos.GetIndexersResponse{IndexersById: indexer_protos.MakeProtoInfos(versions)}
	return ret, nil
}

func (srv *indexerServicer) StartReindex(req *indexer_protos.StartReindexRequest, stream indexer_protos.Indexer_StartReindexServer) error {
	ctx := stream.Context()
	if err := validateCtx(ctx); err != nil {
		return err
	}
	if err := srv.validateReindexReq(req); err != nil {
		return err
	}

	sendUpdate := func(m string) { _ = stream.Send(&indexer_protos.StartReindexResponse{Update: m}) }
	err := srv.reindexer.RunUnsafe(ctx, req.IndexerId, sendUpdate)
	if err != nil {
		return internalErr(err, "error running reindex operation")
	}
	return nil
}

func validateCtx(ctx context.Context) error {
	gw := protos.GetClientGateway(ctx)
	if gw != nil {
		return status.Error(codes.PermissionDenied, "gateway identity found")
	}
	return nil
}

func (srv *indexerServicer) validateReindexReq(req *indexer_protos.StartReindexRequest) error {
	if srv.autoEnabled && !req.Force {
		return status.Error(codes.FailedPrecondition, "automatic reindexing is enabled and request didn't override")
	}
	return nil
}
