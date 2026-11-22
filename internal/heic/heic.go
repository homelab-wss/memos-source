// Package heic decodes HEIC/HEIF still images, a format many browsers cannot
// render directly. Decoding is used to convert uploads, thumbnails, and
// existing attachments to a browser-displayable JPEG.
package heic

import (
	"image"
	"io"
	"strings"

	h265heic "github.com/gen2brain/h265/heic"

	"github.com/usememos/memos/internal/imagelimit"
)

const (
	// MIMETypeHEIC is the MIME type of a HEIC image.
	MIMETypeHEIC = "image/heic"
	// MIMETypeHEIF is the MIME type of a HEIF image.
	MIMETypeHEIF = "image/heif"
)

// IsMIME reports whether mimeType is one of the HEIC/HEIF types this package
// decodes.
func IsMIME(mimeType string) bool {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case MIMETypeHEIC, MIMETypeHEIF:
		return true
	default:
		return false
	}
}

// Decode reads a HEIC/HEIF image and applies the container's rotation and
// mirroring transforms, so the result matches what the camera displayed.
// FrameSizeLimit enforces the shared decode budget even when the header could
// not be sized beforehand, so a crafted file cannot allocate beyond it.
func Decode(r io.Reader) (image.Image, error) {
	return h265heic.Decode(r, h265heic.Options{AutoRotate: true, FrameSizeLimit: imagelimit.MaxPixels})
}

// DecodeConfig reads the image dimensions without decoding its pixels.
func DecodeConfig(r io.Reader) (image.Config, error) {
	return h265heic.DecodeConfig(r)
}
