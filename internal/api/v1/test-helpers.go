package v1

import (
	"context"
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc/metadata"

	keeperpb "github.com/talx-hub/gophkeeper/proto/v1/keeper"
)

func ptr[T any](val T) *T {
	return &val
}

type fakeAddServer struct {
	ctx       context.Context
	reqs      []*keeperpb.AddRequest
	i         int
	resp      *keeperpb.AddResponse
	sentClose bool
}

func newFakeAddStream(ctx context.Context, reqs ...*keeperpb.AddRequest) *fakeAddServer {
	return &fakeAddServer{
		ctx:  ctx,
		reqs: reqs,
	}
}

func (f *fakeAddServer) Recv() (*keeperpb.AddRequest, error) {
	if f.reqs == nil {
		return nil, errors.New("[]requests is nil")
	}

	if f.i >= len(f.reqs) {
		return nil, io.EOF
	}
	r := f.reqs[f.i]
	f.i++
	return r, nil
}

func (f *fakeAddServer) SendAndClose(resp *keeperpb.AddResponse) error {
	if f.sentClose {
		return fmt.Errorf("SendAndClose called twice")
	}
	f.sentClose = true
	f.resp = resp
	return nil
}

func (f *fakeAddServer) Context() context.Context { return f.ctx }

func (f *fakeAddServer) SetHeader(_ metadata.MD) error {
	return nil
}
func (f *fakeAddServer) SendHeader(_ metadata.MD) error {
	return nil
}
func (f *fakeAddServer) SetTrailer(_ metadata.MD) {
}

func (f *fakeAddServer) SendMsg(_ interface{}) error { return nil }

func (f *fakeAddServer) RecvMsg(_ interface{}) error { return nil }

type fakeGetStream struct {
	ctx       context.Context
	responses []*keeperpb.GetResponse
}

func newFakeGetStream(ctx context.Context) *fakeGetStream {
	return &fakeGetStream{
		ctx: ctx,
	}
}

func (f *fakeGetStream) Send(response *keeperpb.GetResponse) error {
	f.responses = append(f.responses, response)
	return nil
}

func (f *fakeGetStream) SetHeader(md metadata.MD) error {
	//TODO implement me
	panic("implement me")
}

func (f *fakeGetStream) SendHeader(md metadata.MD) error {
	//TODO implement me
	panic("implement me")
}

func (f *fakeGetStream) SetTrailer(md metadata.MD) {
	//TODO implement me
	panic("implement me")
}

func (f *fakeGetStream) Context() context.Context {
	return f.ctx
}

func (f *fakeGetStream) SendMsg(m any) error {
	//TODO implement me
	panic("implement me")
}

func (f *fakeGetStream) RecvMsg(m any) error {
	//TODO implement me
	panic("implement me")
}
