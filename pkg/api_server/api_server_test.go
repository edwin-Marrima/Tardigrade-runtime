package api_server

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeWritableFS(t *testing.T) {
	tests := []struct {
		name      string
		imgPath   func(t *testing.T) string
		sizeInMbs int64
		ctx       func() (context.Context, context.CancelFunc)
		linuxOnly bool
		wantErr   bool
	}{
		{
			name: "creates and formats image successfully",
			imgPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "disk.ext4")
			},
			sizeInMbs: 32,
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			linuxOnly: true,
			wantErr:   false,
		},
		{
			name: "output file has correct size",
			imgPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "disk.ext4")
			},
			sizeInMbs: 32,
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			linuxOnly: true,
			wantErr:   false,
		},
		{
			name: "fails when output directory does not exist",
			imgPath: func(t *testing.T) string {
				return "/nonexistent/directory/disk.ext4"
			},
			sizeInMbs: 32,
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			wantErr: true,
		},
		{
			name: "fails when context is already cancelled",
			imgPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "disk.ext4")
			},
			sizeInMbs: 32,
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			wantErr: true,
		},
		{
			name: "fails when size is zero — mkfs.ext4 rejects empty image",
			imgPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "disk.ext4")
			},
			sizeInMbs: 0,
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			linuxOnly: true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.linuxOnly && runtime.GOOS != "linux" {
				t.Skip("requires Linux (mkfs.ext4)")
			}

			ctx, cancel := tt.ctx()
			defer cancel()

			imgPath := tt.imgPath(t)
			err := makeWritableFS(ctx, imgPath, tt.sizeInMbs)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			info, statErr := os.Stat(imgPath)
			require.NoError(t, statErr)
			assert.Equal(t, tt.sizeInMbs*1024*1024, info.Size())
		})
	}
}
