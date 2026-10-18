// Package storage issues presigned, direct-to-R2 upload URLs for
// candidate photos and branding logos. The browser uploads the file
// straight to Cloudflare R2 using the presigned URL - the file itself
// never passes through this server, only a short-lived signed request
// does. What gets stored everywhere else (candidates.photo_url,
// elections.logo_url, system_settings.logo_url) is just the resulting
// public URL, exactly as it already was for the pasted-URL fields this
// replaces - no schema change needed.
package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"backend/internal/config"
)

var ErrNotConfigured = errors.New("file uploads aren't configured on this server")
var ErrUnsupportedType = errors.New("unsupported file type")

const presignTTL = 5 * time.Minute

// Kind namespaces the object key and doubles as the allowlist of what a
// caller may upload - an arbitrary caller-supplied path would let someone
// write anywhere in the bucket.
type Kind string

const (
	KindCandidatePhoto Kind = "candidate-photos"
	KindLogo           Kind = "logos"
)

var allowedKinds = map[Kind]bool{
	KindCandidatePhoto: true,
	KindLogo:           true,
}

var allowedContentTypes = map[string]string{
	"image/png":     "png",
	"image/jpeg":    "jpg",
	"image/webp":    "webp",
	"image/gif":     "gif",
	"image/svg+xml": "svg",
}

type Service struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

// NewService returns nil when R2 isn't configured - callers check for a
// nil service the same way they'd check an error, so an unconfigured
// deployment degrades to "uploads unavailable" rather than failing to
// start at all.
func NewService(cfg *config.Config) *Service {
	if cfg.R2AccountID == "" || cfg.R2AccessKeyID == "" || cfg.R2SecretKey == "" || cfg.R2Bucket == "" || cfg.R2PublicBaseURL == "" {
		return nil
	}

	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2AccountID)),
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.R2AccessKeyID, cfg.R2SecretKey, ""),
	})

	return &Service{client: client, bucket: cfg.R2Bucket, publicURL: cfg.R2PublicBaseURL}
}

type PresignedUpload struct {
	UploadURL string `json:"uploadUrl"`
	PublicURL string `json:"publicUrl"`
}

// PresignUpload generates a one-time PUT URL for the browser to upload
// directly to. The object key is server-generated (random, never the
// caller's filename), so there's no path-traversal or overwrite risk from
// a crafted name.
func (s *Service) PresignUpload(ctx context.Context, kind Kind, contentType string) (*PresignedUpload, error) {
	if s == nil {
		return nil, ErrNotConfigured
	}
	if !allowedKinds[kind] {
		return nil, ErrUnsupportedType
	}
	ext, ok := allowedContentTypes[contentType]
	if !ok {
		return nil, ErrUnsupportedType
	}

	name, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generating object key: %w", err)
	}
	key := fmt.Sprintf("%s/%s.%s", kind, name, ext)

	presignClient := s3.NewPresignClient(s.client, s3.WithPresignExpires(presignTTL))
	req, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("presigning upload: %w", err)
	}

	return &PresignedUpload{
		UploadURL: req.URL,
		PublicURL: s.publicURL + "/" + key,
	}, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
