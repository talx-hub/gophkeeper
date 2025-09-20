package router

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/internal/service/server/keeper"
)

type StoreRouter struct {
	byType          map[model.DataType]keeper.ObjectRepo
	byLocatorPrefix map[string]keeper.ObjectRepo
}

func New(byType map[model.DataType]keeper.ObjectRepo) *StoreRouter {
	return &StoreRouter{
		byType: byType,
	}
}

func (r *StoreRouter) pickByType(dt model.DataType) (keeper.ObjectRepo, error) {
	if s, ok := r.byType[dt]; ok && s != nil {
		return s, nil
	}
	return nil, errors.New("failed to find storage for data type " + string(dt))
}

func (r *StoreRouter) pickByLocator(loc model.ObjectLocator) (keeper.ObjectRepo, error) {
	const s3StoragePrefix = "s3"
	const postgresStoragePrefix = "pg"

	locStr := string(loc)
	switch {
	case strings.HasPrefix(locStr, s3StoragePrefix):
		return r.byLocatorPrefix[s3StoragePrefix], nil
	case strings.HasPrefix(locStr, postgresStoragePrefix):
		return r.byLocatorPrefix[postgresStoragePrefix], nil
	}
	return nil, errors.New("failed to find storage for locator " + locStr)
}

func (r *StoreRouter) Put(
	ctx context.Context,
	meta *model.Metadata,
	rd io.Reader,
	size int32,
	sha256 []byte,
) (model.ObjectLocator, error) {
	storage, err := r.pickByType(meta.DataType)
	if err != nil {
		//nolint // reason: err from wrapped func
		return "", err
	}

	loc, err := storage.Put(ctx, meta, rd, size, sha256)
	if err != nil {
		return "", fmt.Errorf("storage Put: %w", err)
	}
	return loc, nil
}

func (r *StoreRouter) Get(ctx context.Context, loc model.ObjectLocator) (io.ReadCloser, error) {
	storage, err := r.pickByLocator(loc)
	if err != nil {
		//nolint // reason: err from wrapped func
		return nil, err
	}

	rc, err := storage.Get(ctx, loc)
	if err != nil {
		return nil, fmt.Errorf("storage Get: %w", err)
	}
	return rc, nil
}

func (r *StoreRouter) Delete(ctx context.Context, loc model.ObjectLocator) error {
	storage, err := r.pickByLocator(loc)
	if err != nil {
		//nolint // reason: err from wrapped func
		return err
	}

	if err = storage.Delete(ctx, loc); err != nil {
		return fmt.Errorf("storage Delete: %w", err)
	}
	return nil
}
