package handler

import (
	"encoding/json"
	"net/http"
	"pieces-os-go/internal/model"
)

func ListModels(w http.ResponseWriter, r *http.Request) {
	models := make([]model.Model, 0, len(model.SupportedModels))
	for _, model := range model.SupportedModels {
		models = append(models, model)
	}

	response := model.ModelsResponse{
		Object: "list",
		Data:   models,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
