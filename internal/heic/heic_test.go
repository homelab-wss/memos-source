package heic

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/usememos/memos/internal/testutil"
)

func TestIsMIME(t *testing.T) {
	for _, mimeType := range []string{"image/heic", "IMAGE/HEIC", " image/heif "} {
		require.True(t, IsMIME(mimeType), mimeType)
	}
	for _, mimeType := range []string{"", "image/jpeg", "image/heicx", "application/octet-stream"} {
		require.False(t, IsMIME(mimeType), mimeType)
	}
}

func TestDecode(t *testing.T) {
	blob := testutil.BuildHEIC(48, 32)

	config, err := DecodeConfig(bytes.NewReader(blob))
	require.NoError(t, err)
	require.Equal(t, 48, config.Width)
	require.Equal(t, 32, config.Height)

	img, err := Decode(bytes.NewReader(blob))
	require.NoError(t, err)
	require.Equal(t, 48, img.Bounds().Dx())
	require.Equal(t, 32, img.Bounds().Dy())
}
