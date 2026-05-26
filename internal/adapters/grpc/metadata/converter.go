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

	var chunkDescriptor *model.ChunkDescriptor
	if cm := m.GetChunkDescriptor(); cm != nil {
		chunkDescriptor = &model.ChunkDescriptor{
			Offset: cm.GetOffset(),
			CRC32C: cm.GetCrc32C(),
			Last:   cm.GetLast(),
		}
	}

	return &model.Metadata{
		ID:              model.DataID(m.GetId()),
		DataType:        dt,
		Name:            m.GetName(),
		Description:     m.GetDescription(),
		CreatedAt:       created,
		ChunkDescriptor: chunkDescriptor,
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

	var cd *metadatapb.ChunkDescriptor
	if x := m.ChunkDescriptor; x != nil {
		cd = &metadatapb.ChunkDescriptor{
			Offset: &x.Offset,
			Last:   &x.Last,
			Crc32C: &x.CRC32C,
		}
	}

	tempID := string(m.ID)
	return &metadatapb.Metadata{
		DataType:        &pdt,
		Id:              &tempID,
		Name:            &m.Name,
		Description:     &m.Description,
		CreatedAt:       timestamppb.New(m.CreatedAt),
		ChunkDescriptor: cd,
	}
}
