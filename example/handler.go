package example

import (
	"context"
	"fmt"
	"github.com/NYCU-SDC/summer/pkg/handler"
	"github.com/NYCU-SDC/summer/pkg/log"
	"github.com/NYCU-SDC/summer/pkg/problem"
	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"net/http"
)

type Store interface {
	GetAll(ctx context.Context) ([]Comment, error)
	GetById(ctx context.Context, id uuid.UUID) (Comment, error)
	GetByPost(ctx context.Context, postId uuid.UUID) ([]Comment, error)
	Create(ctx context.Context, arg CreateRequest) (Comment, error)
}

type Handler struct {
	logger    *zap.Logger
	tracer    trace.Tracer
	validator *validator.Validate
	store     Store
}

func (h *Handler) CreateHandler(w http.ResponseWriter, r *http.Request) {
	traceCtx, span := h.tracer.Start(r.Context(), "CreateCommentEndpoint")
	defer span.End()
	logger := log.WithContext(traceCtx, h.logger)

	// Parse and validate requestBody body
	var req CreateRequest
	err := handler.ParseAndValidateRequestBody(traceCtx, h.validator, r, &req)
	if err != nil {
		logger.Error("Error decoding requestBody body", zap.Error(err), zap.Any("body", r.Body))
		problem.WriteError(traceCtx, w, err, logger)
		return
	}

	postID := r.PathValue("post_id")

	// Verify and transform PostID to UUID
	id, err := handler.ParseUUID(postID)
	if err != nil {
		logger.Error("Error parsing UUID", zap.Error(err), zap.String("post_id", postID))
		problem.WriteError(traceCtx, w, fmt.Errorf("%w: %v", problem.ErrInvalidUUID, err), logger)
		return
	}
	req.PostID = id

	// Convert AuthorId to UUID
	u, err := jwt.GetUserFromContext(r.Context())
	if err != nil {
		logger.DPanic("Can't find user in context, this should never happen")
		problem.WriteError(traceCtx, w, err, logger)
	}
	authorId, err := internal.ParseUUID(u.ID)
	if err != nil {
		logger.Error("Error getting author id from context", zap.Error(err), zap.String("author_id", u.ID))
		problem.WriteError(traceCtx, w, err, logger)
		return
	}
	req.AuthorID = authorId

	comment, err := h.store.Create(traceCtx, req)
	if err != nil {
		logger.Error("Error creating comment", zap.Error(err))
		problem.WriteError(traceCtx, w, err, logger)
		return
	}

	// Convert comment to Response
	response := GenerateResponse(comment)

	// Write response
	internal.WriteJSONResponse(w, http.StatusOK, response)
}
