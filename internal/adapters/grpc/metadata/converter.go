package metadata

import (
	"errors"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/talx-hub/gophkeeper/internal/model"
	"github.com/talx-hub/gophkeeper/proto/v1"
)

func FromProtoMetadata(m *metadatapb.Metadata) (*model.Metadata, error) {
	var dataType model.DataType
	switch m.GetDataType() {
	case metadatapb.Metadata_DATA_TYPE_AUTH:
		dataType = model.DataTypeAuthenticationCredentials
	case metadatapb.Metadata_DATA_TYPE_CARD:
		dataType = model.DataTypeCard
	case metadatapb.Metadata_DATA_TYPE_BINARY:
		dataType = model.DataTypeBinary
	default:
		return nil, errors.New("unknown metadata type")
	}

	return &model.Metadata{
		DataType:    dataType,
		Name:        m.GetName(),
		Description: m.GetDescription(),
		CreatedAt:   m.GetCreatedAt().AsTime(),
		TotalSize:   m.GetTotalSize(),
	}, nil
}

func ToProtoMetadata(m *model.Metadata) *metadatapb.Metadata {
	var protoDataType metadatapb.Metadata_DataType
	switch m.DataType {
	case model.DataTypeAuthenticationCredentials:
		protoDataType = metadatapb.Metadata_DATA_TYPE_AUTH
	case model.DataTypeCard:
		protoDataType = metadatapb.Metadata_DATA_TYPE_CARD
	case model.DataTypeBinary:
		protoDataType = metadatapb.Metadata_DATA_TYPE_BINARY
	default:
		protoDataType = metadatapb.Metadata_DATA_TYPE_UNSPECIFIED
	}

	int64Ptr := func(id int64) *int64 {
		return &id
	}
	uint64Ptr := func(size uint64) *uint64 {
		return &size
	}

	return &metadatapb.Metadata{
		DataType:    &protoDataType,
		Id:          int64Ptr(int64(m.ID)),
		Name:        &m.Name,
		Description: &m.Description,
		CreatedAt:   timestamppb.New(m.CreatedAt),
		TotalSize:   uint64Ptr(m.TotalSize),
	}
}
