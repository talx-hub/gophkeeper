package v1

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	keeperpb "github.com/talx-hub/gophkeeper/proto/v1/keeper"
)

type KeeperService struct {
	keeperpb.UnimplementedKeeperServer
}

func (s *KeeperService) Sync(*keeperpb.SyncRequest, grpc.ServerStreamingServer[keeperpb.SyncResponse],
) error {
	return status.Errorf(codes.Unimplemented, "method Sync not implemented")
}
func (s *KeeperService) Add(context.Context, *keeperpb.AddRequest,
) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Add not implemented")
}
func (s *KeeperService) List(context.Context, *keeperpb.ListRequest,
) (*keeperpb.ListResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method List not implemented")
}
func (s *KeeperService) Get(context.Context, *keeperpb.GetRequest,
) (*keeperpb.GetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Get not implemented")
}
func (s *KeeperService) Delete(context.Context, *keeperpb.DeleteRequest,
) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}
