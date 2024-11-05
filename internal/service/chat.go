package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"pieces-os-go/internal/config"
	"pieces-os-go/internal/model"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChatService struct {
	grpcService *GRPCService
	config      *config.Config
}

func NewChatService(cfg *config.Config) *ChatService {
	if cfg == nil {
		panic("config cannot be nil")
	}

	grpcService := NewGRPCService(cfg)
	if grpcService == nil {
		panic("failed to create gRPC service")
	}

	return &ChatService{
		grpcService: grpcService,
		config:      cfg,
	}
}

func (s *ChatService) CreateCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	var resp *model.ChatCompletionResponse
	var lastErr error

	// 使用指数退避重试策略
	for i := 0; i < s.config.MaxRetries; i++ {
		resp, lastErr = s.grpcService.SendCompletion(ctx, req)

		// 如果成功或遇到不可重试的错误,直接返回
		if lastErr == nil || !s.shouldRetry(lastErr) {
			return resp, lastErr
		}

		// 计算退避时间
		backoff := time.Duration(1<<uint(i)) * time.Second

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			log.Printf("重试 RPC 调用，尝试次数: %d, 错误: %v", i+1, lastErr)
			continue
		}
	}

	return nil, fmt.Errorf("达到最大重试次数: %v", lastErr)
}

func (s *ChatService) CreateCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan *model.ChatCompletionStreamResponse, <-chan error) {
	responses := make(chan *model.ChatCompletionStreamResponse)
	errors := make(chan error, 1)

	go func() {
		defer close(responses)
		defer close(errors)

		stream, err := s.grpcService.SendCompletionStream(ctx, req)
		if err != nil {
			errors <- err
			return
		}

		for resp := range stream {
			select {
			case <-ctx.Done():
				errors <- ctx.Err()
				return
			case responses <- resp:
			}
		}
	}()

	return responses, errors
}

func (s *ChatService) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// 定义可重试的状态码
	retryableStatusCodes := map[codes.Code]bool{
		codes.Unavailable:       true,
		codes.ResourceExhausted: true,
		codes.DeadlineExceeded:  true,
		codes.Internal:          true,
		codes.Unknown:           true,
		codes.Canceled:          true,
	}

	// 检查 gRPC 状态码
	if st, ok := status.FromError(err); ok {
		return retryableStatusCodes[st.Code()]
	}

	// 检查上下文错误
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) {
		return true
	}

	// 检查网络错误
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}

	return false
}
