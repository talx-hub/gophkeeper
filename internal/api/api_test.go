package api

import (
	"context"
	"errors"
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
const certsFixturesDir = "./certs-fixtures"

type DummyDBManager struct {
}

func (m *DummyDBManager) GetPool() (*pgxpool.Pool, error) {
	return &pgxpool.Pool{}, nil
}

type BrokenDBManager struct {
}

func (m *BrokenDBManager) GetPool() (*pgxpool.Pool, error) {
	return nil, errors.New("expected error")
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
				&config.Config{RunAddr: tt.address, CertsDir: certsFixturesDir},
				&DummyDBManager{},
				slog.Default())
			assert.NotNil(t, s)
		})
	}
}

func TestServer_Setup(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"init pool fail", true},
		{"load credentials fail", true},
		{"ok", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(
				&config.Config{RunAddr: "::"},
				&DummyDBManager{},
				slog.Default())
			if tt.name == "init pool fail" {
				s.dbManager = &BrokenDBManager{}
			}
			if tt.name == "ok" {
				s.dbManager = &DummyDBManager{}
				s.cfg.CertsDir = certsFixturesDir
			}
			err := s.Setup()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_Serve(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"invalid: wrong IP", "w.r.o.ng", true},
		{"invalid: no Port #1", "localhost", true},
		{"invalid: wrong Port #1", "localhost:WRONG", true},
		{"invalid: wrong Port #2", "localhost:99999", true},
		{"valid address", "localhost:", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(
				&config.Config{RunAddr: tt.address, CertsDir: certsFixturesDir},
				&DummyDBManager{},
				slog.Default())
			err := s.Setup()
			require.NoError(t, err)

			time.AfterFunc(startServerTO, func() {
				_ = s.Stop(context.Background())
			})
			err = s.Serve()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
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
				&config.Config{RunAddr: "localhost:", CertsDir: certsFixturesDir},
				&DummyDBManager{},
				slog.Default())
			require.NotNil(t, s)

			err := s.Setup()

			require.NoError(t, err)
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()

				err := s.Serve()
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
