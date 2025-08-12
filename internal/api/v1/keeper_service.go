package v1

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/talx-hub/gophkeeper/proto/v1/keeper"
)

type KeeperService struct {
	keeper.UnimplementedKeeperServer
}

func (s *KeeperService) Sync(*keeper.SyncRequest, grpc.ServerStreamingServer[keeper.SyncResponse],
) error {
	return status.Errorf(codes.Unimplemented, "method Sync not implemented")
}
func (s *KeeperService) Add(context.Context, *keeper.AddRequest,
) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Add not implemented")
}
func (s *KeeperService) List(context.Context, *keeper.ListRequest,
) (*keeper.ListResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method List not implemented")
}
func (s *KeeperService) Get(context.Context, *keeper.GetRequest,
) (*keeper.GetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Get not implemented")
}
func (s *KeeperService) Delete(context.Context, *keeper.DeleteRequest,
) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}
