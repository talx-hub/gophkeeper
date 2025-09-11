package v1

import (
	"context"
	"errors"
	"io"

	"google.golang.org/grpc/metadata"

	keeperpb "github.com/talx-hub/gophkeeper/proto/v1/keeper"
)

func ptr[T any](val T) *T {
	return &val
}

type fakeAddStream struct {
	//nolint:containedctx // reason: real stream contains CTX too, need it
	ctx       context.Context
	resp      *keeperpb.AddResponse
	reqs      []*keeperpb.AddRequest
	i         int
	sentClose bool
}

func newFakeAddStream(ctx context.Context, reqs ...*keeperpb.AddRequest) *fakeAddStream {
	return &fakeAddStream{
		ctx:  ctx,
		reqs: reqs,
	}
}

func (f *fakeAddStream) Recv() (*keeperpb.AddRequest, error) {
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

func (f *fakeAddStream) SendAndClose(resp *keeperpb.AddResponse) error {
	if f.sentClose {
		return errors.New("SendAndClose called twice")
	}
	f.sentClose = true
	f.resp = resp
	return nil
}

func (f *fakeAddStream) Context() context.Context { return f.ctx }

func (f *fakeAddStream) SetHeader(_ metadata.MD) error {
	return nil
}
func (f *fakeAddStream) SendHeader(_ metadata.MD) error {
	return nil
}
func (f *fakeAddStream) SetTrailer(_ metadata.MD) {
}

func (f *fakeAddStream) SendMsg(_ interface{}) error { return nil }

func (f *fakeAddStream) RecvMsg(_ interface{}) error { return nil }

type fakeGetStream struct {
	//nolint:containedctx // reason: real stream contains CTX too, need it
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

func (f *fakeGetStream) SetHeader(_ metadata.MD) error {
	return nil
}

func (f *fakeGetStream) SendHeader(_ metadata.MD) error {
	return nil
}

func (f *fakeGetStream) SetTrailer(_ metadata.MD) {
}

func (f *fakeGetStream) Context() context.Context {
	return f.ctx
}

func (f *fakeGetStream) SendMsg(_ any) error {
	return nil
}

func (f *fakeGetStream) RecvMsg(_ any) error {
	return nil
}

type fakeSyncStream struct {
	//nolint:containedctx // reason: real stream contains CTX too, need it
	ctx       context.Context
	responses []*keeperpb.SyncResponse
}

func newFakeSyncStream(ctx context.Context) *fakeSyncStream {
	return &fakeSyncStream{
		ctx: ctx,
	}
}

func (f *fakeSyncStream) Send(response *keeperpb.SyncResponse) error {
	f.responses = append(f.responses, response)
	return nil
}

func (f *fakeSyncStream) SetHeader(_ metadata.MD) error {
	return nil
}

func (f *fakeSyncStream) SendHeader(_ metadata.MD) error {
	return nil
}

func (f *fakeSyncStream) SetTrailer(_ metadata.MD) {
}

func (f *fakeSyncStream) Context() context.Context {
	return f.ctx
}

func (f *fakeSyncStream) SendMsg(_ any) error {
	return nil
}

func (f *fakeSyncStream) RecvMsg(_ any) error {
	return nil
}
