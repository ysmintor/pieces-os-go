package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"pieces-os-go/internal/config"
	"pieces-os-go/internal/model"
	"pieces-os-go/internal/service"
)

type ChatHandler struct {
	chatService *service.ChatService
	config      *config.Config
}

func NewChatHandler(cfg *config.Config) *ChatHandler {
	return &ChatHandler{
		chatService: service.NewChatService(cfg),
		config:      cfg,
	}
}

func (h *ChatHandler) HandleCompletion(w http.ResponseWriter, r *http.Request) {
	var req model.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, model.NewAPIError(model.ErrInvalidRequest, err.Error(), http.StatusBadRequest))
		return
	}

	if req.Stream {
		h.handleStreamCompletion(w, r, &req)
		return
	}

	h.handleNormalCompletion(w, r, &req)
}

// 处理流式请求
func (h *ChatHandler) handleStreamCompletion(w http.ResponseWriter, r *http.Request, req *model.ChatCompletionRequest) {
	// 设置 SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok || flusher == nil {
		writeError(w, model.NewAPIError(model.ErrInternalError, "Streaming not supported", http.StatusInternalServerError))
		return
	}

	stream, errChan := h.chatService.CreateCompletionStream(r.Context(), req)

	for {
		select {
		case <-r.Context().Done():
			// 处理连接关闭和上下文取消
			log.Printf("Connection closed or context cancelled: %v", r.Context().Err())
			return

		case err := <-errChan:
			if err != nil {
				if apiErr, ok := err.(*model.APIError); ok {
					if err := writeSSEError(w, flusher, apiErr.Code, apiErr.Message); err != nil {
						log.Printf("Failed to write SSE error: %v", err)
					}
				} else {
					if err := writeSSEError(w, flusher, model.ErrInternalError, err.Error()); err != nil {
						log.Printf("Failed to write SSE error: %v", err)
					}
				}
				return
			}

		case chunk, ok := <-stream:
			if !ok {
				// 流结束前检查上下文状态
				if r.Context().Err() != nil {
					return
				}
				// 流正常结束
				if err := writeSSEChunk(w, flusher, "[DONE]"); err != nil {
					log.Printf("Failed to write final SSE chunk: %v", err)
				}
				return
			}

			if chunk != nil {
				// 写入数据前检查上下文状态
				if r.Context().Err() != nil {
					return
				}
				if err := writeSSEChunk(w, flusher, chunk); err != nil {
					log.Printf("Failed to write SSE chunk: %v", err)
					return
				}
			}
		}
	}
}

// 处理普通请求
func (h *ChatHandler) handleNormalCompletion(w http.ResponseWriter, r *http.Request, req *model.ChatCompletionRequest) {
	resp, err := h.chatService.CreateCompletion(r.Context(), req)
	if err != nil {
		if apiErr, ok := err.(*model.APIError); ok {
			writeError(w, apiErr)
		} else {
			writeError(w, model.NewAPIError(model.ErrInternalError, err.Error(), http.StatusInternalServerError))
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// 辅助函数
func writeError(w http.ResponseWriter, err *model.APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": err.Message,
			"type":    "error",
			"code":    err.Code,
		},
	})
}

// 修改辅助函数，增加错误返回
func writeSSEError(w http.ResponseWriter, flusher http.Flusher, code model.ErrorCode, message string) error {
	if flusher == nil {
		return fmt.Errorf("flusher is nil")
	}

	errResp := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"code":    code,
		},
	}
	data, err := json.Marshal(errResp)
	if err != nil {
		return fmt.Errorf("failed to marshal error response: %v", err)
	}

	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return fmt.Errorf("failed to write error message: %v", err)
	}

	flusher.Flush()
	return nil
}

func writeSSEChunk(w http.ResponseWriter, flusher http.Flusher, chunk interface{}) error {
	if flusher == nil {
		return fmt.Errorf("flusher is nil, cannot write SSE chunk")
	}

	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	if err != nil {
		return err
	}

	flusher.Flush()
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// WithModel 包装处理函数，为请求预设模型
func WithModel(h http.HandlerFunc, modelName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		// 如果请求体中没有指定模型或模型不合法，使用URL中的模型
		if req.Model == "" || !model.IsModelSupported(model.NormalizeModelName(req.Model)) {
			req.Model = modelName
		}

		// 重新编码请求体
		newBody, err := json.Marshal(req)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "无法处理请求"})
			return
		}

		// 创建新的请求
		r.Body = io.NopCloser(bytes.NewBuffer(newBody))
		h.ServeHTTP(w, r)
	}
}
