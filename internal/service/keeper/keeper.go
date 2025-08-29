// Package keeper реализует прикладную бизнес-логику
// для работы с хранилищем зашифрованных данных пользователя.
// Сервис инкапсулирует операции добавления, получения,
// удаления и синхронизации объектов, абстрагируя доступ
// к объектному стораджу и репозиторию метаданных через интерфейсы.
package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"github.com/talx-hub/gophkeeper/internal/model"
)

// Service реализует операции управления зашифрованными данными:
// запись, чтение, удаление и синхронизацию.
// Сервис обращается к репозиторию объектов (ObjectRepo)
// и репозиторию метаданных (MetadataRepo).
type Service struct {
	objectRepo   ObjectRepo
	metadataRepo MetadataRepo
}

// NewService возвращает новый экземпляр Service,
// используя указанные реализации хранилищ объектов и метаданных.
func NewService(objectRepo ObjectRepo, metadataRepo MetadataRepo) *Service {
	return &Service{
		objectRepo:   objectRepo,
		metadataRepo: metadataRepo,
	}
}

// ObjectRepo абстрагирует доступ к хранилищу объектов (хранилище не обязательно реляционное).
// Интерфейс поддерживает операции записи, чтения и удаления бинарных данных.
type ObjectRepo interface {
	// Put сохраняет объект, считанный из r, в хранилище.
	// Size задаёт ожидаемую длину (0 — если неизвестна).
	// sha256, если передан (len==32), используется для проверки целостности
	// записанных данных.
	// Возвращает локатор объекта и фактические сведения о записи.
	Put(ctx context.Context, meta *model.Metadata, r io.Reader, size uint64, sha256 []byte,
	) (ObjectLocator, ObjectInfo, error)

	// Get возвращает поток для чтения объекта и сведения о нём.
	Get(ctx context.Context, loc ObjectLocator) (io.ReadCloser, ObjectInfo, error)

	// Delete удаляет объект по указанному локатору.
	Delete(ctx context.Context, loc ObjectLocator) error
}

// MetadataRepo абстрагирует работу с метаданными в реляционном хранилище.
// Содержит операции создания, получения, выборки и удаления.
type MetadataRepo interface {
	// Create сохраняет метаданные и связывает их с локатором объекта и сведениями о нём.
	Create(ctx context.Context, meta *model.Metadata, info ObjectInfo, loc ObjectLocator,
	) (model.DataID, error)

	// Get возвращает метаданные и локатор объекта по userID и DataID.
	Get(ctx context.Context, userID model.UserID, id model.DataID) (model.Metadata, ObjectLocator, error)

	// ListByUser возвращает список всех метаданных пользователя вместе с локаторами объектов.
	ListByUser(ctx context.Context, userID model.UserID) ([]MetaLoc, error)

	// Delete удаляет метаданные по userID и DataID и возвращает локатор объекта.
	Delete(ctx context.Context, userID model.UserID, id model.DataID) (ObjectLocator, error)
}

// ObjectInfo описывает фактические характеристики объекта,
// полученные после записи в сторадж.
type ObjectInfo struct {
	Size   uint64
	SHA256 [32]byte
}

// ObjectLocator — непрозрачный идентификатор местоположения объекта
// в объектном хранилище (например, "pg://...", "s3://...").
type ObjectLocator string

// MetaLoc связывает доменные метаданные и локатор объекта.
// Используется при выборках списков объектов.
type MetaLoc struct {
	Meta    model.Metadata
	Locator ObjectLocator
}

// StreamCallback вызывается сервисом при выдаче объекта наружу.
// Для малых объектов sealed содержит все данные,
// для больших в будущем будет вызвано несколько раз по чанкам.
type StreamCallback func(meta *model.Metadata, sealed []byte) error

// SyncMode задаёт режим синхронизации.
type SyncMode int

const (
	SyncModeShort SyncMode = iota + 1 // Только малые по объему данные
	SyncModeFull                      // Данные из БД + S3
)

// AddSealed сохраняет зашифрованный объект sealed и соответствующие метаданные.
// Функция вычисляет контрольную сумму, сохраняет объект в ObjectRepo,
// затем создаёт запись в MetadataRepo. В случае ошибки метаданные не создаются,
// а объект удаляется из ObjectRepo.
func (s *Service) AddSealed(ctx context.Context,
	userID model.UserID,
	meta *model.Metadata,
	sealed []byte,
) (model.DataID, error) {
	if userID == "" {
		return 0, errors.New("empty userID")
	}
	if meta == nil {
		return 0, errors.New("nil metadata")
	}
	if len(sealed) == 0 {
		return 0, errors.New("empty sealed_data")
	}
	switch meta.DataType {
	case model.DataTypeAuthenticationCredentials, model.DataTypeCard, model.DataTypeBinary:
	default:
		return 0, fmt.Errorf("unsupported data type: %v", meta.DataType)
	}
	meta.UserID = userID

	ctx1, cancel1 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel1()
	sum := sha256.Sum256(sealed)
	loc, info, err := s.objectRepo.Put(ctx1, meta, bytes.NewReader(sealed), uint64(len(sealed)), sum[:])
	if err != nil {
		return 0, fmt.Errorf("failed to put sealed data to object repo: %w", err)
	}

	ctx2, cancel2 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel2()
	id, err := s.metadataRepo.Create(ctx2, meta, info, loc)
	if err != nil {
		if errDelete := s.objectRepo.Delete(ctx2, loc); errDelete != nil {
			err = errors.Join(err, errDelete)
		}
		return 0, fmt.Errorf("failed to put metadata to meta repo: %w", err)
	}

	return id, nil
}

// GetSealed получает объект по его идентификатору и вызывает cb
// для передачи метаданных и зашифрованных данных вызывающему коду.
// Используется при gRPC-стриминге или других потоковых интерфейсах.
func (s *Service) GetSealed(ctx context.Context,
	userID model.UserID,
	id model.DataID,
	cb StreamCallback,
) (err error) {
	if userID == "" {
		return errors.New("empty userID")
	}
	if id == 0 {
		return errors.New("empty id")
	}

	ctx1, cancel1 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel1()
	meta, loc, err := s.metadataRepo.Get(ctx1, userID, id)
	if err != nil {
		return fmt.Errorf("failed to get metadata from repo: %w", err)
	}

	sealed, err := s.getSealedHelper(ctx, loc)
	if err != nil {
		//nolint:wrapcheck // err from helper function
		return err
	}

	if err := cb(&meta, sealed); err != nil {
		return fmt.Errorf("callback failed: %w", err)
	}
	return nil
}

// List возвращает список всех метаданных пользователя вместе с локаторами объектов.
func (s *Service) List(ctx context.Context, userID model.UserID) ([]MetaLoc, error) {
	if userID == "" {
		return nil, errors.New("empty userID")
	}
	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()
	metaLoc, err := s.metadataRepo.ListByUser(ctxTO, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list metadata from repo: %w", err)
	}

	return metaLoc, nil
}

// Delete удаляет объект и его метаданные по идентификатору.
func (s *Service) Delete(ctx context.Context, userID model.UserID, id model.DataID) error {
	if userID == "" {
		return errors.New("empty userID")
	}
	if id == 0 {
		return errors.New("empty id")
	}

	ctx1, cancel1 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel1()
	loc, err := s.metadataRepo.Delete(ctx1, userID, id)
	if err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	ctx2, cancel2 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel2()
	if err := s.objectRepo.Delete(ctx2, loc); err != nil {
		return fmt.Errorf("failed to delete data: %w", err)
	}

	return nil
}

// Sync выполняет синхронизацию данных пользователя.
func (s *Service) Sync(ctx context.Context,
	userID model.UserID, mode SyncMode, cb StreamCallback,
) error {
	if userID == "" {
		return errors.New("empty userID")
	}

	ctx1, cancel1 := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel1()
	metaLocs, err := s.metadataRepo.ListByUser(ctx1, userID)
	if err != nil {
		return fmt.Errorf("repo.ListMetadata method failed: %w", err)
	}

	for i := range metaLocs {
		select {
		case <-ctx.Done():
			return fmt.Errorf("metadata run through failed: %w", ctx.Err())
		default:
		}

		switch mode {
		case SyncModeShort:
			sealed, err := s.getSealedHelper(ctx, metaLocs[i].Locator)
			if err != nil {
				//nolint:wrapcheck // err from helper function
				return err
			}

			if err := cb(&metaLocs[i].Meta, sealed); err != nil {
				return fmt.Errorf(
					"metadata and data grpc stream write callback failed: %w", err)
			}
		case SyncModeFull:
			return errors.New("full sync mode isn't implemented yet")
		default:
			return errors.New("unknown sync mode")
		}
	}

	return nil
}

// getSealedHelper — вспомогательная функция для загрузки объекта из ObjectRepo.
// Считывает все данные в память и возвращает их как []byte.
func (s *Service) getSealedHelper(ctx context.Context, loc ObjectLocator) (sealed []byte, err error) {
	ctxTO, cancel := context.WithTimeout(ctx, model.RepoOperationTO)
	defer cancel()

	rc, _, err := s.objectRepo.Get(ctxTO, loc)
	if err != nil {
		return nil, fmt.Errorf("objectRepo.Get method failed: %w", err)
	}
	defer func() {
		err = errors.Join(err, rc.Close())
	}()

	sealed, err = io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return sealed, nil
}
