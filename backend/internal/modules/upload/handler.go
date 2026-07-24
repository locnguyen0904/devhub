package upload

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/locnguyen0904/devhub/backend/internal/modules/auth"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
	"github.com/locnguyen0904/devhub/backend/internal/platform/storage"
)

var (
	errUnsupportedType = httpx.Invalid("Unsupported image type", map[string]string{
		"content_type": "must be image/png, image/jpeg, image/webp or image/gif",
	})
	errTooLarge = httpx.Invalid("Image too large", map[string]string{
		"size_bytes": "must be between 1 byte and 5 MB",
	})
)

// Handler serves the presign endpoint.
type Handler struct {
	svc *Service
}

func newHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// PresignInput is the upload request: what the browser is about to send.
type PresignInput struct {
	Body struct {
		ContentType string `json:"content_type" example:"image/png"`
		SizeBytes   int64  `json:"size_bytes" example:"482913"`
	}
}

// PresignOutput carries the URL to PUT to and the URL the image will live at.
type PresignOutput struct {
	Body struct {
		UploadURL string `json:"upload_url" doc:"PUT the file here with the same Content-Type"`
		PublicURL string `json:"public_url" doc:"Where the image is served once uploaded"`
		ExpiresIn int    `json:"expires_in" doc:"Seconds the upload URL stays valid"`
	}
}

func (h *Handler) presign(ctx context.Context, in *PresignInput) (*PresignOutput, error) {
	actor, err := auth.RequireIdentity(ctx)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}

	presigned, err := h.svc.Presign(ctx, actor.UserID, in.Body.ContentType, in.Body.SizeBytes)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}

	out := &PresignOutput{}
	out.Body.UploadURL = presigned.UploadURL
	out.Body.PublicURL = presigned.PublicURL
	out.Body.ExpiresIn = presigned.ExpiresIn
	return out, nil
}

// Module wires the upload module.
type Module struct {
	handler *Handler
}

// NewModule builds the module from the storage service.
func NewModule(store presigner) *Module {
	return &Module{handler: newHandler(New(store))}
}

// Register mounts the presign endpoint. It requires a bearer token, enforced by
// auth.RequireIdentity inside the handler.
func (m *Module) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "presignUpload",
		Method:      http.MethodPost,
		Path:        "/api/v1/uploads/presign",
		Summary:     "Get a presigned URL to upload an image directly to storage",
		Tags:        []string{"uploads"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, m.handler.presign)
}

var _ presigner = (*storage.Storage)(nil)
