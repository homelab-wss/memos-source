package api

import (
	"bytes"
	"context"
	"image/jpeg"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/usememos/memos/internal/testutil"
	apipb "github.com/usememos/memos/proto/gen/api"
	"github.com/usememos/memos/store"
)

func TestCreateAttachmentConvertsHEICToJPEG(t *testing.T) {
	ctx := context.Background()
	svc := newIntegrationService(t)
	owner := createSpaceTestUser(ctx, t, svc, "heic-owner", store.RoleUser)

	content := testutil.BuildHEIC(64, 64)
	attachment, err := svc.CreateAttachment(userCtx(ctx, owner.ID), &apipb.CreateAttachmentRequest{
		Attachment: &apipb.Attachment{
			Filename: "IMG_0001.HEIC",
			Type:     "image/heic",
			Content:  content,
		},
	})
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", attachment.Type)
	require.Equal(t, "IMG_0001.jpg", attachment.Filename)

	uid, err := ExtractAttachmentUIDFromName(attachment.Name)
	require.NoError(t, err)
	row, err := svc.Store.GetAttachment(ctx, &store.FindAttachment{UID: &uid})
	require.NoError(t, err)
	require.NotNil(t, row)
	require.Equal(t, "image/jpeg", row.Type)
	require.Equal(t, "IMG_0001.jpg", row.Filename)

	stored, err := os.ReadFile(row.Reference)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(stored, []byte{0xff, 0xd8}), "stored content should be a JPEG")
	img, err := jpeg.Decode(bytes.NewReader(stored))
	require.NoError(t, err)
	require.Equal(t, 64, img.Bounds().Dx())
	require.Equal(t, 64, img.Bounds().Dy())
}
