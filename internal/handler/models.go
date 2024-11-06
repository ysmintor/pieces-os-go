package handler

import (
	"encoding/json"
	"net/http"
	"pieces-os-go/internal/model"
)

func ListModels(w http.ResponseWriter, r *http.Request) {
	// 检查请求方法
	if r.Method != http.MethodGet {
		writeError(w, model.NewAPIError(model.ErrMethodNotAllowed, "Only GET method is allowed", http.StatusMethodNotAllowed))
		return
	}

	models := make([]model.Model, 0, len(model.SupportedModels))
	for _, model := range model.SupportedModels {
		models = append(models, model)
	}

	response := model.ModelsResponse{
		Object: "list",
		Data:   models,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		writeError(w, model.NewAPIError(model.ErrInternalError, "Failed to encode response", http.StatusInternalServerError))
		return
	}
}
