package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"my-currency-service/gateway/internal/middleware"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// WriteError writes JSONError with current HTTP code
func WriteError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	var (
		code    = codes.Internal
		message = "interrnal server error"
	)

	if s, ok := status.FromError(err); ok {
		code = s.Code()
		message = s.Message()
	} else if errors.Is(err, errInvalidQuery) {
		code = codes.InvalidArgument
		message = err.Error()
	}

	httpStatus := grpcCodeToHTTP(code)
	// incapsulate internal detail
	if code == codes.Internal {
		log.ErrorContext(r.Context(), "internal error", slog.Any("error", err))
		message = "internal server error"
	}

	reqID := middleware.RequestIDFromContext(r.Context())
	payload := ErrorResponse{Error: ErrorDetail{
		Code:      code.String(),
		Message:   message,
		RequestID: reqID,
	}}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(payload)
}

func grpcCodeToHTTP(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.Unimplemented:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

var errInvalidQuery = errors.New("invalid query")
