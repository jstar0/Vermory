package webchat

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"vermory/internal/runtime"
)

const maxRequestBodyBytes int64 = 1 << 20

type Handler struct {
	service  *runtime.ConversationService
	defaults *runtime.GlobalDefaultsService
	mux      *http.ServeMux
}

func NewHandler(service *runtime.ConversationService, defaultServices ...*runtime.GlobalDefaultsService) http.Handler {
	var defaults *runtime.GlobalDefaultsService
	if len(defaultServices) > 0 {
		defaults = defaultServices[0]
	}
	handler := &Handler{service: service, defaults: defaults, mux: http.NewServeMux()}
	handler.mux.HandleFunc("POST /v1/chat/turn", handler.chatTurn)
	handler.mux.HandleFunc("POST /v1/memories/confirm", handler.confirmMemory)
	handler.mux.HandleFunc("POST /v1/memories/correct", handler.correctMemory)
	handler.mux.HandleFunc("POST /v1/memories/forget", handler.forgetMemory)
	handler.mux.HandleFunc("GET /v1/conversations/inspect", handler.inspectConversation)
	handler.mux.HandleFunc("GET /v1/defaults", handler.inspectDefaults)
	handler.mux.HandleFunc("POST /v1/defaults/set", handler.setDefault)
	handler.mux.HandleFunc("POST /v1/defaults/correct", handler.correctDefault)
	handler.mux.HandleFunc("POST /v1/defaults/forget", handler.forgetDefault)
	return handler
}

func (h *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	h.mux.ServeHTTP(response, request)
}

type conversationInput struct {
	OperationID string `json:"operation_id"`
	Channel     string `json:"channel"`
	ThreadID    string `json:"thread_id"`
}

type chatTurnInput struct {
	conversationInput
	Message string `json:"message"`
}

type confirmMemoryInput struct {
	conversationInput
	ObservationID string `json:"observation_id"`
}

type correctMemoryInput struct {
	conversationInput
	MemoryID string `json:"memory_id"`
	Content  string `json:"content"`
}

type forgetMemoryInput struct {
	conversationInput
	MemoryID string `json:"memory_id"`
}

type setDefaultInput struct {
	OperationID string `json:"operation_id"`
	Key         string `json:"key"`
	Content     string `json:"content"`
}

type correctDefaultInput struct {
	OperationID string `json:"operation_id"`
	MemoryID    string `json:"memory_id"`
	Content     string `json:"content"`
}

type forgetDefaultInput struct {
	OperationID string `json:"operation_id"`
	MemoryID    string `json:"memory_id"`
}

func (h *Handler) chatTurn(response http.ResponseWriter, request *http.Request) {
	var input chatTurnInput
	if !decodeRequestJSON(response, request, &input) {
		return
	}
	receipt, err := h.service.Chat(request.Context(), runtime.ChatTurnRequest{
		OperationID: input.OperationID,
		Anchor:      runtime.ConversationAnchor{Channel: input.Channel, ThreadID: input.ThreadID},
		Message:     input.Message,
	})
	if err != nil {
		writeServiceError(response, err)
		return
	}
	status := http.StatusOK
	if receipt.Status == runtime.ChatTurnFailed {
		status = http.StatusBadGateway
	}
	writeJSON(response, status, receipt)
}

func (h *Handler) confirmMemory(response http.ResponseWriter, request *http.Request) {
	var input confirmMemoryInput
	if !decodeRequestJSON(response, request, &input) {
		return
	}
	receipt, err := h.service.Confirm(request.Context(), runtime.ConfirmConversationMemoryRequest{
		OperationID:   input.OperationID,
		Anchor:        runtime.ConversationAnchor{Channel: input.Channel, ThreadID: input.ThreadID},
		ObservationID: input.ObservationID,
	})
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, receipt)
}

func (h *Handler) correctMemory(response http.ResponseWriter, request *http.Request) {
	var input correctMemoryInput
	if !decodeRequestJSON(response, request, &input) {
		return
	}
	receipt, err := h.service.Correct(request.Context(), runtime.CorrectConversationMemoryRequest{
		OperationID: input.OperationID,
		Anchor:      runtime.ConversationAnchor{Channel: input.Channel, ThreadID: input.ThreadID},
		MemoryID:    input.MemoryID,
		Content:     input.Content,
	})
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, receipt)
}

func (h *Handler) forgetMemory(response http.ResponseWriter, request *http.Request) {
	var input forgetMemoryInput
	if !decodeRequestJSON(response, request, &input) {
		return
	}
	receipt, err := h.service.Forget(request.Context(), runtime.ForgetConversationMemoryRequest{
		OperationID: input.OperationID,
		Anchor:      runtime.ConversationAnchor{Channel: input.Channel, ThreadID: input.ThreadID},
		MemoryID:    input.MemoryID,
	})
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, receipt)
}

func (h *Handler) inspectConversation(response http.ResponseWriter, request *http.Request) {
	inspection, err := h.service.Inspect(request.Context(), runtime.ConversationAnchor{
		Channel:  request.URL.Query().Get("channel"),
		ThreadID: request.URL.Query().Get("thread_id"),
	})
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, inspection)
}

func (h *Handler) inspectDefaults(response http.ResponseWriter, request *http.Request) {
	if !h.requireDefaults(response) {
		return
	}
	inspection, err := h.defaults.Inspect(request.Context())
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, inspection)
}

func (h *Handler) setDefault(response http.ResponseWriter, request *http.Request) {
	if !h.requireDefaults(response) {
		return
	}
	var input setDefaultInput
	if !decodeRequestJSON(response, request, &input) {
		return
	}
	receipt, err := h.defaults.Set(request.Context(), runtime.SetGlobalDefaultRequest{
		OperationID: input.OperationID,
		Key:         input.Key,
		Content:     input.Content,
	})
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, receipt)
}

func (h *Handler) correctDefault(response http.ResponseWriter, request *http.Request) {
	if !h.requireDefaults(response) {
		return
	}
	var input correctDefaultInput
	if !decodeRequestJSON(response, request, &input) {
		return
	}
	receipt, err := h.defaults.Correct(request.Context(), runtime.CorrectGlobalDefaultRequest{
		OperationID: input.OperationID,
		MemoryID:    input.MemoryID,
		Content:     input.Content,
	})
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, receipt)
}

func (h *Handler) forgetDefault(response http.ResponseWriter, request *http.Request) {
	if !h.requireDefaults(response) {
		return
	}
	var input forgetDefaultInput
	if !decodeRequestJSON(response, request, &input) {
		return
	}
	receipt, err := h.defaults.Forget(request.Context(), runtime.ForgetGlobalDefaultRequest{
		OperationID: input.OperationID,
		MemoryID:    input.MemoryID,
	})
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, receipt)
}

func (h *Handler) requireDefaults(response http.ResponseWriter) bool {
	if h.defaults != nil {
		return true
	}
	writeError(response, http.StatusInternalServerError, "internal_error", "request could not be completed")
	return false
}

func decodeRequestJSON(response http.ResponseWriter, request *http.Request, target any) bool {
	if mediaType := strings.TrimSpace(strings.Split(request.Header.Get("Content-Type"), ";")[0]); mediaType != "application/json" {
		writeError(response, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return false
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(response, http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large")
			return false
		}
		writeError(response, http.StatusBadRequest, "invalid_json", "request body must contain one valid JSON object")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(response, http.StatusBadRequest, "invalid_json", "request body must contain exactly one JSON object")
		return false
	}
	return true
}

func writeServiceError(response http.ResponseWriter, err error) {
	message := err.Error()
	switch {
	case strings.Contains(message, "does not exist"):
		writeError(response, http.StatusNotFound, "not_found", "conversation does not exist")
	case isSafeClientError(message):
		writeError(response, http.StatusBadRequest, "invalid_request", message)
	default:
		writeError(response, http.StatusInternalServerError, "internal_error", "request could not be completed")
	}
}

func isSafeClientError(message string) bool {
	for _, fragment := range []string{
		" is required",
		" is too long",
		" does not belong",
		" cannot become memory",
		"must be an active fact",
		"already bound",
		"already has an active value",
		"key must use",
	} {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

func writeError(response http.ResponseWriter, status int, code, message string) {
	writeJSON(response, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	if err := json.NewEncoder(response).Encode(value); err != nil {
		_, _ = fmt.Fprintln(response, `{"error":{"code":"encoding_error","message":"response could not be encoded"}}`)
	}
}
