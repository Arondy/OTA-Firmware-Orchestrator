package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/transport/http/dto"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const maxRequestBodyBytes = 1 << 20

func DecodeJSONBody(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, v any) bool {
	body := http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(v)
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		logger.Warnw("json body exceeds the limit", "error", err)
		WriteError(w, logger, http.StatusRequestEntityTooLarge, fmt.Sprintf("request body exceeds limit in %d bytes", maxRequestBodyBytes))
		return false
	}
	if err == io.EOF {
		logger.Warnw("empty json", "error", err)
		WriteError(w, logger, http.StatusBadRequest, "request body is required")
		return false
	} else if strings.HasPrefix(err.Error(), "json: unknown field") {
		logger.Warnw("unknown field in request body", "error", err)
		msg := fmt.Sprintf("unknown field in request body: %s", strings.TrimPrefix(err.Error(), "json: unknown field "))
		WriteError(w, logger, http.StatusBadRequest, msg)
		return false
	} else if err != nil {
		logger.Warnw("failed to decode request body", "error", err)
		WriteError(w, logger, http.StatusBadRequest, "invalid JSON")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		logger.Warnw("symbols after json", "error", err)
		WriteError(w, logger, http.StatusBadRequest, "request body must contain a single JSON object")
		return false
	}
	return true
}

func ValidateRequest(w http.ResponseWriter, logger *zap.SugaredLogger, req any) bool {
	err := dto.Validate.Struct(req)
	if err == nil {
		return true
	}

	valErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		logger.Errorw("failed to validate request fields", "error", err)
		WriteInternalServerError(w, logger)
		return false
	}

	messages := dto.FormatValidation(valErrs)
	logger.Warnw("request didn't pass validation", "fields", messages)
	response := map[string]any{"error": "request didn't pass validation", "fields": messages}

	WriteJSON(w, logger, http.StatusBadRequest, response)
	return false
}

func ParseUUIDFromPath(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger) (id uuid.UUID, ok bool) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warnw("incorrect uuid", "error", err)
		WriteError(w, logger, http.StatusBadRequest, "invalid UUID in path")
		return uuid.Nil, false
	}

	return id, true
}

func WriteJSON(w http.ResponseWriter, logger *zap.SugaredLogger, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Warnw("failed to write JSON response", "error", err)
	}
}

func WriteError(w http.ResponseWriter, logger *zap.SugaredLogger, status int, message string) {
	WriteJSON(w, logger, status, map[string]string{"error": message})
}

func WriteInternalServerError(w http.ResponseWriter, logger *zap.SugaredLogger) {
	WriteError(w, logger, http.StatusInternalServerError, "internal server error")
}
