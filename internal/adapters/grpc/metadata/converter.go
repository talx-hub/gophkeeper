package metadata

import (
	"errors"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/talx-hub/gophkeeper/internal/model"
	metadatapb "github.com/talx-hub/gophkeeper/proto/v1/metadata"
)

func FromProtoMetadata(m *metadatapb.Metadata) (*model.Metadata, error) {
	if m == nil {
		return nil, errors.New("nil proto metadata")
	}

	var dt model.DataType
	switch m.GetDataType() {
	case metadatapb.Metadata_DATA_TYPE_AUTH:
		dt = model.DataTypeAuthenticationCredentials
	case metadatapb.Metadata_DATA_TYPE_CARD:
		dt = model.DataTypeCard
	case metadatapb.Metadata_DATA_TYPE_BINARY:
		dt = model.DataTypeBinary
	default:
		return nil, errors.New("unknown metadata type")
	}

	var created time.Time
	if ts := m.GetCreatedAt(); ts != nil {
		created = ts.AsTime()
	}

	var chunkMeta *model.ChunkMetadata
	if cm := m.GetChunkMetadata(); cm != nil {
		chunkMeta = &model.ChunkMetadata{
			Offset:       cm.GetOffset(),
			ObjectSize:   cm.GetObjectSize(),
			Last:         cm.GetLast(),
			CRC32C:       cm.GetCrc32C(),
			ObjectSHA256: cm.GetObjectSha256(),
		}
	}

	return &model.Metadata{
		ID:            model.DataID(m.GetId()),
		DataType:      dt,
		Name:          m.GetName(),
		Description:   m.GetDescription(),
		CreatedAt:     created,
		ChunkMetadata: chunkMeta,
	}, nil
}

func ToProtoMetadata(m *model.Metadata) *metadatapb.Metadata {
	if m == nil {
		return nil
	}

	var pdt metadatapb.Metadata_DataType
	switch m.DataType {
	case model.DataTypeAuthenticationCredentials:
		pdt = metadatapb.Metadata_DATA_TYPE_AUTH
	case model.DataTypeCard:
		pdt = metadatapb.Metadata_DATA_TYPE_CARD
	case model.DataTypeBinary:
		pdt = metadatapb.Metadata_DATA_TYPE_BINARY
	default:
		pdt = metadatapb.Metadata_DATA_TYPE_UNSPECIFIED
	}

	var cm *metadatapb.ChunkMetadata
	if x := m.ChunkMetadata; x != nil {
		cm = &metadatapb.ChunkMetadata{
			Offset:       &x.Offset,
			ObjectSize:   &x.ObjectSize,
			Last:         &x.Last,
			Crc32C:       &x.CRC32C,
			ObjectSha256: x.ObjectSHA256,
		}
	}

	return &metadatapb.Metadata{
		DataType:      &pdt,
		Id:            ptrInt64(int64(m.ID)),
		Name:          &m.Name,
		Description:   &m.Description,
		CreatedAt:     timestamppb.New(m.CreatedAt),
		ChunkMetadata: cm,
	}
}

func ptrInt64(val int64) *int64 {
	return &val
}
