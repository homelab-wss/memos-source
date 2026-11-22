package api

import (
	"bytes"
	"context"
	"image"
	"io"
	"net/http"

	"github.com/disintegration/imaging"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/usememos/memos/internal/heic"
	"github.com/usememos/memos/internal/imagelimit"
	apipb "github.com/usememos/memos/proto/gen/api"
	storepb "github.com/usememos/memos/proto/gen/store"
)

func validateClientMotionMedia(motion *apipb.MotionMedia, attachmentUID string) (*storepb.MotionMedia, error) {
	if motion == nil {
		return nil, nil
	}

	if motion.Family != apipb.MotionMediaFamily_APPLE_LIVE_PHOTO {
		return nil, status.Errorf(codes.InvalidArgument, "only Apple Live Photo motion metadata can be provided by clients")
	}
	if motion.Role != apipb.MotionMediaRole_STILL && motion.Role != apipb.MotionMediaRole_VIDEO {
		return nil, status.Errorf(codes.InvalidArgument, "invalid Apple Live Photo motion role")
	}

	storeMotion := convertMotionMediaToStore(motion)
	if storeMotion.GroupId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "motion media group_id is required")
	}
	if storeMotion.Family == storepb.MotionMediaFamily_ANDROID_MOTION_PHOTO && storeMotion.GroupId == "" {
		storeMotion.GroupId = attachmentUID
	}

	return storeMotion, nil
}

// shouldStripExif checks if the MIME type is an image format that may contain EXIF metadata.
// Returns true for formats like JPEG, TIFF, WebP, HEIC, and HEIF which commonly contain
// privacy-sensitive metadata such as GPS coordinates, camera settings, and device information.
func shouldStripExif(mimeType string) bool {
	return exifCapableImageTypes[mimeType]
}

// shouldStripExifContent reports whether the content must be stripped of
// EXIF metadata, judging by the declared type and by sniffing the leading
// bytes. The source is rewound afterwards.
func shouldStripExifContent(source io.ReadSeeker, declaredType string) (bool, error) {
	if shouldStripExif(declaredType) {
		return true, nil
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return false, errors.Wrap(err, "failed to rewind attachment content")
	}
	header := make([]byte, 512)
	n, err := io.ReadFull(source, header)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return false, errors.Wrap(err, "failed to read attachment header")
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return false, errors.Wrap(err, "failed to rewind attachment content")
	}
	sniffed, ok := normalizeMimeType(http.DetectContentType(header[:n]))
	return ok && shouldStripExif(sniffed), nil
}

func (s *APIService) acquireImageProcessingSlot(ctx context.Context) (func(), error) {
	if s.imageProcessingSemaphore == nil {
		return func() {}, nil
	}
	if err := s.imageProcessingSemaphore.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	return func() {
		s.imageProcessingSemaphore.Release(1)
	}, nil
}

func validateImageReaderPixelCount(reader io.Reader) error {
	return imagelimit.CheckReader(reader)
}

// validateImageReaderPixelCountForMIME bounds the declared pixel count for the
// formats whose header the standard image registry cannot size. HEIC/HEIF
// headers are read by their own decoder; a header it cannot parse is left to
// the full decode to reject.
func validateImageReaderPixelCountForMIME(reader io.Reader, mimeType string) error {
	if heic.IsMIME(mimeType) {
		config, err := heic.DecodeConfig(reader)
		if err != nil {
			return nil //nolint:nilerr // unsupported/corrupt headers defer to the decoder
		}
		return imagelimit.CheckDimensions(config.Width, config.Height)
	}
	return validateImageReaderPixelCount(reader)
}

// stripImageExif removes EXIF metadata from image files by decoding and re-encoding them.
// This prevents exposure of sensitive metadata such as GPS location, camera details, and timestamps.
//
// The function preserves the correct image orientation by applying EXIF orientation tags
// during decoding before stripping all metadata. Images are re-encoded with high quality
// to minimize visual degradation. The re-encoded output is returned in memory; its size
// is bounded by imagelimit.MaxPixels, which the decoder already has to hold.
//
// Supported formats:
//   - JPEG/JPG: Re-encoded as JPEG with quality 95
//   - PNG: Re-encoded as PNG (lossless)
//   - TIFF/WebP/HEIC/HEIF: Re-encoded as JPEG with quality 95
//
// Returns the cleaned image data without any EXIF metadata, or an error if processing fails.
func stripImageExif(source io.ReadSeeker, mimeType string) ([]byte, error) {
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Wrap(err, "failed to rewind image")
	}
	if err := validateImageReaderPixelCountForMIME(source, mimeType); err != nil {
		return nil, err
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Wrap(err, "failed to rewind image")
	}
	var (
		img image.Image
		err error
	)
	if heic.IsMIME(mimeType) {
		img, err = heic.Decode(source)
	} else {
		img, err = imaging.Decode(source, imaging.AutoOrientation(true))
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode image")
	}
	var buf bytes.Buffer
	if mimeType == "image/png" {
		err = imaging.Encode(&buf, img, imaging.PNG)
	} else {
		err = imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(defaultJPEGQuality))
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode image")
	}
	return buf.Bytes(), nil
}
