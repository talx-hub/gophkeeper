package api

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gophkeeper/pkg/config"
)

const startServerTO = 100 * time.Millisecond

type DummyDBManager struct {
}

func (m *DummyDBManager) GetPool() (*pgxpool.Pool, error) {
	return &pgxpool.Pool{}, nil
}

func TestNewServer(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantNil bool
	}{
		{"success", "localhost:8888", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(
				&config.Config{RunAddr: tt.address},
				&DummyDBManager{},
				slog.Default())
			assert.NotNil(t, s)
		})
	}
}

func TestServer_Start(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"valid address", "localhost:", false},
		{"invalid: wrong IP", "1:", true},
		{"invalid: no Port #1", "localhost", true},
		{"invalid: wrong Port #1", "localhost:WRONG", true},
		{"invalid: wrong Port #2", "localhost:99999", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(
				&config.Config{RunAddr: tt.address},
				&DummyDBManager{},
				slog.Default())

			wg := &sync.WaitGroup{}
			wg.Add(1)
			time.AfterFunc(startServerTO, func() {
				defer wg.Done()

				err := s.Stop(context.Background())
				require.NoError(t, err)
			})
			err := s.Start()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			wg.Wait()
		})
	}
}

func TestServer_Stop(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"successful close", false},
		{"deadline exceeded", true},
	}

	var ctx context.Context
	var cancel context.CancelFunc
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(
				&config.Config{RunAddr: "localhost:"},
				&DummyDBManager{},
				slog.Default())
			require.NotNil(t, s)

			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()

				err := s.Start()
				require.NoError(t, err)
			}()
			time.Sleep(startServerTO)

			if tt.wantErr {
				ctx, cancel = context.WithCancel(context.Background())
				cancel()

				err := s.Stop(ctx)
				assert.Error(t, err)
			} else {
				ctx, cancel = context.WithTimeout(context.Background(), time.Second)
				err := s.Stop(ctx)
				cancel()
				assert.NoError(t, err)
			}
			wg.Wait()
		})
	}
}
