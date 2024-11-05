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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if req.Stream {
		// 设置 SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// 创建一个done channel来控制流
		done := make(chan bool)
		defer close(done)

		// 调用服务层的流式方法
		stream, errChan := h.chatService.CreateCompletionStream(r.Context(), &req)

		go func() {
			defer func() {
				done <- true
			}()

			for chunk := range stream {
				select {
				case err := <-errChan:
					if err != nil {
						log.Printf("Stream error: %v", err)
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
						return
					}
				default:
					streamResp := chunk

					data, err := json.Marshal(streamResp)
					if err != nil {
						log.Printf("Error marshaling response: %v", err)
						return
					}

					// 写入 SSE 格式数据
					_, err = fmt.Fprintf(w, "data: %s\n\n", data)
					if err != nil {
						log.Printf("Error writing response: %v", err)
						return
					}
					w.(http.Flusher).Flush()
				}
			}
		}()

		// 等待流完成
		<-done

		// 发送结束标记
		fmt.Fprintf(w, "data: [DONE]\n\n")
		w.(http.Flusher).Flush()
		return
	}

	resp, err := h.chatService.CreateCompletion(r.Context(), &req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
