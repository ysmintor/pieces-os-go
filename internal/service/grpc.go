package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"pieces-os-go/internal/config"
	"pieces-os-go/internal/model"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gptpb "pieces-os-go/pkg/proto/gpt"
	vertexpb "pieces-os-go/pkg/proto/vertex"
	"pieces-os-go/pkg/tokenizer"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

type GRPCService struct {
	config     *config.Config
	vertexPool *ConnectionPool
	gptPool    *ConnectionPool
	connMutex  sync.RWMutex
}

type ConnectionPool struct {
	connections chan *grpc.ClientConn
	addr        string
	minSize     int   // 最小连接数
	maxSize     int   // 最大连接数
	currentSize int32 // 当前连接数
	mu          sync.RWMutex
}

func NewGRPCService(cfg *config.Config) *GRPCService {
	service := &GRPCService{
		config: cfg,
	}

	// 初始化连接池
	if cfg.VertexGRPCAddr != "" {
		service.vertexPool = newConnectionPool(cfg.VertexGRPCAddr, 5, 20) // 最小5个,最大20个
	}
	if cfg.GPTGRPCAddr != "" {
		service.gptPool = newConnectionPool(cfg.GPTGRPCAddr, 5, 20)
	}

	return service
}

func newConnectionPool(addr string, minSize, maxSize int) *ConnectionPool {
	if minSize <= 0 {
		minSize = 5 // 默认最小连接数
	}
	if maxSize <= 0 || maxSize < minSize {
		maxSize = minSize * 2 // 默认最大连接数
	}

	pool := &ConnectionPool{
		connections: make(chan *grpc.ClientConn, maxSize),
		addr:        addr,
		minSize:     minSize,
		maxSize:     maxSize,
		currentSize: 0,
	}

	// 预创建最小数量的连接
	for i := 0; i < minSize; i++ {
		if conn, err := createNewConnection(addr); err == nil {
			pool.connections <- conn
			atomic.AddInt32(&pool.currentSize, 1)
		}
	}

	// 启动自动扩缩容goroutine
	go pool.autoScale()

	return pool
}

func (s *GRPCService) getConnection(addr string) (*grpc.ClientConn, error) {
	s.connMutex.RLock()
	defer s.connMutex.RUnlock()

	var pool *ConnectionPool

	if addr == s.config.VertexGRPCAddr {
		pool = s.vertexPool
	} else if addr == s.config.GPTGRPCAddr {
		pool = s.gptPool
	}

	if pool == nil {
		return nil, fmt.Errorf("no pool available for address: %s", addr)
	}

	// 尝试从池中获取连接
	select {
	case conn := <-pool.connections:
		if conn.GetState() != connectivity.Shutdown {
			return conn, nil
		}
		// 连接已关闭,创建新连接
		atomic.AddInt32(&pool.currentSize, -1)
	default:
		// 池为空但未达到最大值时创建新连接
		if atomic.LoadInt32(&pool.currentSize) < int32(pool.maxSize) {
			if conn, err := createNewConnection(addr); err == nil {
				atomic.AddInt32(&pool.currentSize, 1)
				return conn, nil
			}
		}
	}

	// 等待可用连接
	select {
	case conn := <-pool.connections:
		return conn, nil
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("connection pool timeout")
	}
}

func createNewConnection(addr string) (*grpc.ClientConn, error) {
	// 创建 TLS 凭证
	creds := credentials.NewClientTLSFromCert(nil, "")

	// 修改 keepalive 参数
	kacp := keepalive.ClientParameters{
		Time:                30 * time.Second, // 增加到 30 秒
		Timeout:             10 * time.Second, // 增加到 10 秒
		PermitWithoutStream: false,            // 改为 false，只在有活动流时发送 ping
	}

	// 使用 NewClient 创建连接
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(kacp),
		grpc.WithInitialWindowSize(1<<20),     // 1MB
		grpc.WithInitialConnWindowSize(1<<20), // 1MB
		grpc.WithUserAgent("dart-grpc/2.0.0"), // 添加 User-Agent 设置
	)
	if err != nil {
		return nil, fmt.Errorf("connection error")
	}

	return conn, nil
}

func (s *GRPCService) SendCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	// 空值检查
	if req == nil {
		return nil, model.NewAPIError(model.ErrInvalidRequest, "request cannot be nil", http.StatusBadRequest)
	}
	if s.config == nil {
		return nil, model.NewAPIError(model.ErrInternalError, "service configuration is not initialized", http.StatusInternalServerError)
	}

	// 在开始时标准化模型名称
	originalModel := req.Model
	req.Model = model.NormalizeModelName(req.Model)

	// 验证模型是否支持，如果不支持则尝试使用默认模型
	if !model.IsModelSupported(req.Model) {
		if s.config.DefaultModel == "" {
			return nil, model.NewAPIError(
				model.ErrModelNotFound,
				fmt.Sprintf("Model '%s' does not exist", originalModel),
				http.StatusNotFound,
			)
		}
		req.Model = s.config.DefaultModel
	}

	var conn *grpc.ClientConn
	var err error

	if model.IsGPTModel(req.Model) {
		conn, err = s.getConnection(s.config.GPTGRPCAddr)
		if err != nil {
			return nil, fmt.Errorf("service unavailable: %v", err)
		}
		defer s.gptPool.returnConnection(conn)

		// 使用buildGRPCRequest构建请求
		grpcReq := buildGRPCRequest(req).(*gptpb.Request)
		if grpcReq == nil {
			return nil, fmt.Errorf("failed to build GPT request")
		}

		client := gptpb.NewGPTInferenceServiceClient(conn)
		resp, err := client.Predict(ctx, grpcReq)
		if err != nil {
			return nil, fmt.Errorf("request failed: %v", err)
		}

		// 添加空值检查
		if resp == nil {
			return nil, fmt.Errorf("received nil response")
		}

		// 检查响应状态码
		if resp.ResponseCode != 200 && resp.ResponseCode != 0 {
			return nil, fmt.Errorf("service error: response code %d", resp.ResponseCode)
		}

		// 使用tokenizer计算token数量
		promptTokens := tokenizer.NumTokensFromMessages(req.Messages, req.Model)

		// 增加响应结构的完整性检查
		if resp.Body == nil || resp.Body.MessageWarpper == nil ||
			resp.Body.MessageWarpper.Message == nil {
			return nil, fmt.Errorf("invalid response structure")
		}

		content := resp.Body.MessageWarpper.Message.Message
		if content == "" {
			return nil, fmt.Errorf("empty response content")
		}

		completionTokens := tokenizer.NumTokensFromMessage(&model.ChatMessage{
			Role:    model.RoleAssistant,
			Content: content,
		}, req.Model)

		// 转换为 OpenAI 格式响应
		response := &model.ChatCompletionResponse{
			ID:      resp.Body.Id,
			Object:  model.ObjectChatCompletion,
			Created: int64(resp.Body.Time),
			Model:   originalModel,
			Choices: []*model.Choice{
				{
					Message: &model.ChatMessage{
						Role:    model.RoleAssistant,
						Content: content,
					},
					Index: 0,
				},
			},
			Usage: &model.Usage{
				PromptTokens:     promptTokens,
				CompletionTokens: completionTokens,
				TotalTokens:      promptTokens + completionTokens,
			},
		}
		return response, nil

	} else {
		conn, err = s.getConnection(s.config.VertexGRPCAddr)
		if err != nil {
			return nil, fmt.Errorf("service unavailable: %v", err)
		}
		defer s.vertexPool.returnConnection(conn)

		// 使用buildGRPCRequest构建请求
		grpcReq := buildGRPCRequest(req).(*vertexpb.Requests)
		if grpcReq == nil {
			return nil, fmt.Errorf("failed to build Vertex request")
		}

		client := vertexpb.NewVertexInferenceServiceClient(conn)
		resp, err := client.Predict(ctx, grpcReq)
		if err != nil {
			return nil, fmt.Errorf("request failed: %v", err)
		}

		// 添加空值检查
		if resp == nil {
			return nil, fmt.Errorf("received nil response")
		}

		// 检查响应状态码
		if resp.ResponseCode != 200 && resp.ResponseCode != 0 {
			return nil, fmt.Errorf("service error: response code %d", resp.ResponseCode)
		}

		// 使用tokenizer计算token数量
		params, _ := buildTokenCountParams(req.Messages)
		promptTokens, err := tokenizer.NumTokensFromClaudeMessages(&params)
		if err != nil {
			log.Printf("Error counting prompt tokens: %v", err)
			promptTokens = 0
		}

		// 增加响应结构的完整性检查
		if resp.Args == nil || resp.Args.Args == nil ||
			resp.Args.Args.Args == nil {
			return nil, fmt.Errorf("invalid response structure")
		}

		content := resp.Args.Args.Args.Message
		if content == "" {
			return nil, fmt.Errorf("empty response content")
		}

		completionTokens, err := tokenizer.CountTokens(content)
		if err != nil {
			log.Printf("Error counting completion tokens: %v", err)
			completionTokens = 0
		} else {
			completionTokens += 3
		}

		// 转换为 OpenAI 格式响应
		response := &model.ChatCompletionResponse{
			ID:      generateChatID(),
			Object:  model.ObjectChatCompletion,
			Created: time.Now().Unix(),
			Model:   originalModel,
			Choices: []*model.Choice{
				{
					Message: &model.ChatMessage{
						Role:    model.RoleAssistant,
						Content: content,
					},
					Index: 0,
				},
			},
			Usage: &model.Usage{
				PromptTokens:     promptTokens,
				CompletionTokens: completionTokens,
				TotalTokens:      promptTokens + completionTokens,
			},
		}
		return response, nil
	}
}

func (s *GRPCService) SendCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan *model.ChatCompletionStreamResponse, error) {
	// 空值检查
	if req == nil {
		return nil, model.NewAPIError(model.ErrInvalidRequest, "request cannot be nil", http.StatusBadRequest)
	}
	if s.config == nil {
		return nil, model.NewAPIError(model.ErrInternalError, "service configuration is not initialized", http.StatusInternalServerError)
	}

	// 在开始时标准化模型名称
	originalModel := req.Model
	req.Model = model.NormalizeModelName(req.Model)

	// 验证模型是否支持，如果不支持则尝试使用默认模型
	if !model.IsModelSupported(req.Model) {
		if s.config.DefaultModel == "" {
			return nil, model.NewAPIError(
				model.ErrModelNotFound,
				fmt.Sprintf("Model '%s' does not exist", originalModel),
				http.StatusNotFound,
			)
		}
		req.Model = s.config.DefaultModel
	}
	// 验证模型是否支持，如果不支持则尝试使用默认模型
	if !model.IsModelSupported(req.Model) {
		if s.config.DefaultModel == "" {
			return nil, model.NewAPIError(
				model.ErrModelNotFound,
				fmt.Sprintf("Model '%s' does not exist", originalModel),
				http.StatusNotFound,
			)
		}
		req.Model = s.config.DefaultModel
	}

	responseChan := make(chan *model.ChatCompletionStreamResponse)

	if model.IsGPTModel(req.Model) {
		conn, err := s.getConnection(s.config.GPTGRPCAddr)
		if err != nil {
			return nil, fmt.Errorf("service unavailable")
		}
		defer s.gptPool.returnConnection(conn)

		// 使用 buildGRPCRequest 构建请求
		grpcReq := buildGRPCRequest(req).(*gptpb.Request)

		client := gptpb.NewGPTInferenceServiceClient(conn)
		stream, err := client.PredictWithStream(ctx, grpcReq)
		if err != nil {
			return nil, fmt.Errorf("stream request failed")
		}

		go func() {
			defer close(responseChan)

			responseID := generateChatID()
			promptTokens := tokenizer.NumTokensFromMessages(req.Messages, req.Model)
			var completionTokens int
			var fullContent string
			isFirstChunk := true

			// 获取当前时间戳作为默认值
			currentTime := time.Now().Unix()

			for {
				select {
				case <-ctx.Done():
					log.Printf("GPT stream timeout or canceled")
					return
				default:
					resp, err := stream.Recv()
					if err != nil {
						if err != io.EOF {
							log.Printf("GPT stream error: %v", err)
						}
						return
					}

					// 添加空值检查
					if resp == nil {
						log.Printf("Received nil response")
						continue
					}

					// 添加空值检查并使用默认时间
					createdTime := currentTime
					if resp.Body != nil {
						createdTime = int64(resp.Body.Time)
					}

					// 处理 204 响应码
					if resp.ResponseCode == 204 {
						// 发送最终响应前的空值检查
						if resp.Body != nil && resp.Body.MessageWarpper != nil &&
							resp.Body.MessageWarpper.Message != nil &&
							resp.Body.MessageWarpper.Message.Message != "" {

							content := resp.Body.MessageWarpper.Message.Message
							fullContent += content

							response := &model.ChatCompletionStreamResponse{
								ID:      responseID,
								Object:  model.ObjectChatCompletionChunk,
								Created: createdTime,
								Model:   originalModel,
								Choices: []*model.ChatCompletionStreamChoice{
									{
										Delta: &model.ChatCompletionStreamDelta{
											Content: content,
										},
										Index: 0,
									},
								},
							}

							if isFirstChunk {
								response.Choices[0].Delta.Role = model.RoleAssistant
								isFirstChunk = false
							}

							select {
							case responseChan <- response:
							case <-ctx.Done():
								return
							}
						}

						// 发送最终响应
						completionTokens = tokenizer.NumTokensFromText(fullContent, req.Model)
						finalResponse := &model.ChatCompletionStreamResponse{
							ID:      responseID,
							Object:  model.ObjectChatCompletionChunk,
							Created: createdTime,
							Model:   originalModel,
							Choices: []*model.ChatCompletionStreamChoice{
								{
									Delta:        &model.ChatCompletionStreamDelta{},
									Index:        0,
									FinishReason: model.FinishReasonStop,
								},
							},
							Usage: &model.Usage{
								PromptTokens:     promptTokens,
								CompletionTokens: completionTokens,
								TotalTokens:      promptTokens + completionTokens,
							},
						}

						select {
						case responseChan <- finalResponse:
						case <-ctx.Done():
							return
						}
						return
					}

					// 处理常规消息
					if resp.Body != nil && resp.Body.MessageWarpper != nil &&
						resp.Body.MessageWarpper.Message != nil &&
						resp.Body.MessageWarpper.Message.Message != "" {

						content := resp.Body.MessageWarpper.Message.Message
						fullContent += content

						response := &model.ChatCompletionStreamResponse{
							ID:      responseID,
							Object:  model.ObjectChatCompletionChunk,
							Created: int64(resp.Body.Time),
							Model:   originalModel,
							Choices: []*model.ChatCompletionStreamChoice{
								{
									Delta: &model.ChatCompletionStreamDelta{
										Content: content,
									},
									Index: 0,
								},
							},
						}

						if isFirstChunk {
							response.Choices[0].Delta.Role = model.RoleAssistant
							isFirstChunk = false
						}

						select {
						case responseChan <- response:
						case <-ctx.Done():
							return
						}
					} else {
						log.Printf("Received incomplete message structure")
						continue
					}
				}
			}
		}()

	} else {
		conn, err := s.getConnection(s.config.VertexGRPCAddr)
		if err != nil {
			return nil, err
		}
		defer s.vertexPool.returnConnection(conn)

		grpcReq := buildGRPCRequest(req).(*vertexpb.Requests)
		client := vertexpb.NewVertexInferenceServiceClient(conn)
		stream, err := client.PredictWithStream(ctx, grpcReq)
		if err != nil {
			return nil, fmt.Errorf("stream request failed")
		}

		go func() {
			defer close(responseChan)

			responseID := generateChatID()
			params, _ := buildTokenCountParams(req.Messages)
			promptTokens, err := tokenizer.NumTokensFromClaudeMessages(&params)
			if err != nil {
				log.Printf("Error counting prompt tokens: %v", err)
				promptTokens = 0
			}
			var completionTokens int
			var fullContent string
			isFirstChunk := true

			for {
				select {
				case <-ctx.Done():
					log.Printf("Stream timeout or canceled")
					return
				default:
					resp, err := stream.Recv()
					if err != nil {
						if err != io.EOF {
							if st, ok := status.FromError(err); ok {
								if st.Code() == codes.Internal && strings.Contains(st.Message(), "RST_STREAM") {
									log.Printf("Stream terminated by RST_STREAM")
								} else {
									log.Printf("Stream error with code %v: %v", st.Code(), st.Message())
								}
							} else {
								log.Printf("Stream error: %v", err)
							}
						}
						// 发送最终响应
						if fullContent != "" {
							if completionTokens, err = tokenizer.CountTokens(fullContent); err != nil {
								log.Printf("Error counting completion tokens: %v", err)
								completionTokens = 0
							} else {
								completionTokens += 3
							}

							select {
							case responseChan <- &model.ChatCompletionStreamResponse{
								ID:      responseID,
								Object:  model.ObjectChatCompletionChunk,
								Created: time.Now().Unix(),
								Model:   originalModel,
								Choices: []*model.ChatCompletionStreamChoice{
									{
										Delta:        &model.ChatCompletionStreamDelta{},
										Index:        0,
										FinishReason: model.FinishReasonStop,
									},
								},
								Usage: &model.Usage{
									PromptTokens:     promptTokens,
									CompletionTokens: completionTokens,
									TotalTokens:      promptTokens + completionTokens,
								},
							}:
							case <-ctx.Done():
								return
							}
						}
						return
					}

					if resp == nil {
						log.Printf("Received nil response")
						continue
					}

					// 处理204响应码
					if resp.ResponseCode == 204 {
						if resp.Args != nil && resp.Args.Args != nil &&
							resp.Args.Args.Args != nil && resp.Args.Args.Args.Message != "" {
							content := resp.Args.Args.Args.Message
							fullContent += content

							select {
							case responseChan <- &model.ChatCompletionStreamResponse{
								ID:      responseID,
								Object:  model.ObjectChatCompletionChunk,
								Created: time.Now().Unix(),
								Model:   originalModel,
								Choices: []*model.ChatCompletionStreamChoice{
									{
										Delta: &model.ChatCompletionStreamDelta{
											Content: content,
										},
										Index: 0,
									},
								},
							}:
							case <-ctx.Done():
								return
							}
						}

						// 发送最终响应
						if completionTokens, err = tokenizer.CountTokens(fullContent); err != nil {
							log.Printf("Error counting completion tokens: %v", err)
							completionTokens = 0
						} else {
							completionTokens += 3
						}

						select {
						case responseChan <- &model.ChatCompletionStreamResponse{
							ID:      responseID,
							Object:  model.ObjectChatCompletionChunk,
							Created: time.Now().Unix(),
							Model:   originalModel,
							Choices: []*model.ChatCompletionStreamChoice{
								{
									Delta:        &model.ChatCompletionStreamDelta{},
									Index:        0,
									FinishReason: model.FinishReasonStop,
								},
							},
							Usage: &model.Usage{
								PromptTokens:     promptTokens,
								CompletionTokens: completionTokens,
								TotalTokens:      promptTokens + completionTokens,
							},
						}:
						case <-ctx.Done():
							return
						}
						return
					}

					// 检查响应结构的完整性
					if resp.Args != nil && resp.Args.Args != nil &&
						resp.Args.Args.Args != nil && resp.Args.Args.Args.Message != "" {
						content := resp.Args.Args.Args.Message
						fullContent += content

						response := &model.ChatCompletionStreamResponse{
							ID:      responseID,
							Object:  model.ObjectChatCompletionChunk,
							Created: time.Now().Unix(),
							Model:   originalModel,
							Choices: []*model.ChatCompletionStreamChoice{
								{
									Delta: &model.ChatCompletionStreamDelta{
										Content: content,
									},
									Index: 0,
								},
							},
						}

						if isFirstChunk {
							response.Choices[0].Delta.Role = model.RoleAssistant
							isFirstChunk = false
						}

						select {
						case responseChan <- response:
						case <-ctx.Done():
							return
						}
					} else {
						log.Printf("Received incomplete message structure")
						continue
					}
				}
			}
		}()
	}

	return responseChan, nil
}

func (s *GRPCService) Close() error {
	if s.vertexPool != nil {
		s.vertexPool.closeAll()
	}
	if s.gptPool != nil {
		s.gptPool.closeAll()
	}
	return nil
}

// 辅助函数
func buildGRPCRequest(req *model.ChatCompletionRequest) interface{} {
	if model.IsGPTModel(req.Model) {
		var systemContent string
		var dialogContent string

		// 处理消息
		for _, msg := range req.Messages {
			if msg.Role == model.RoleSystem {
				systemContent += fmt.Sprintf("system:%s;\r\n", msg.Content)
			} else {
				dialogContent += fmt.Sprintf("%s:%s;\r\n", msg.Role, msg.Content)
			}
		}

		messages := []*gptpb.Message{}
		if systemContent != "" {
			messages = append(messages, &gptpb.Message{
				Role:    0,
				Message: systemContent,
			})
		}
		if dialogContent != "" {
			messages = append(messages, &gptpb.Message{
				Role:    1,
				Message: dialogContent,
			})
		}
		grpcReq := &gptpb.Request{
			Models:      req.Model,
			Messages:    messages,
			Temperature: req.Temperature,
			TopP:        req.TopP,
		}

		// log.Printf("GPT Request: %+v", grpcReq)
		return grpcReq
	} else {
		// 构建 Vertex 请求
		params, system := buildTokenCountParams(req.Messages)
		var conversations []string

		for _, msg := range params.Messages {
			conversations = append(conversations, fmt.Sprintf("%s:%s", msg.Role, msg.Content))
		}

		message := ""
		if len(conversations) > 0 {
			message = strings.Join(conversations, ";\r\n") + ";\r\n"
		}

		if system != "" {
			system = "system:" + system + ";\r\n"
		}

		return &vertexpb.Requests{
			Models: model.NormalizeModelName(req.Model),
			Args: &vertexpb.Args{
				Messages: &vertexpb.Messages{
					Unknown: 1,
					Message: message,
				},
				Rules: system,
			},
		}
	}
}

// 辅助函数用于构建 TokenCountParams
func buildTokenCountParams(messages []model.ChatMessage) (tokenizer.TokenCountParams, string) {
	var systemMessages []string
	var conversations []model.ChatMessage
	var currentMessage model.ChatMessage

	// 先分离系统消息
	for _, msg := range messages {
		if msg.Role == model.RoleSystem {
			systemMessages = append(systemMessages, msg.Content)
		} else {
			// 处理非系统消息
			if currentMessage.Role == "" {
				currentMessage = msg
			} else if currentMessage.Role == msg.Role {
				// 合并相同角色的连续消息
				currentMessage.Content += "\n" + msg.Content
			} else {
				// 角色变化，保存当前消息并开始新消息
				conversations = append(conversations, currentMessage)
				currentMessage = msg
			}
		}
	}

	// 添加最后一条消息
	if currentMessage.Role != "" {
		conversations = append(conversations, currentMessage)
	}

	system := strings.Join(systemMessages, "\n")

	return tokenizer.TokenCountParams{
		Messages: conversations,
		System:   system,
	}, system
}

// 新增一个生成统一格式 ID 的辅助函数
func generateChatID() string {
	// 生成 UUID 并去掉横线，取前 28 位
	id := strings.ReplaceAll(uuid.New().String(), "-", "")[:28]
	return "chatcmpl-" + id
}

// 添加归还连接的方法
func (p *ConnectionPool) returnConnection(conn *grpc.ClientConn) {
	if conn == nil {
		return
	}

	select {
	case p.connections <- conn:
		// 成功归还到池中
	default:
		// 池已满,关闭多余连接
		conn.Close()
	}
}

// 添加关闭所有连接的方法
func (p *ConnectionPool) closeAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for {
		select {
		case conn := <-p.connections:
			conn.Close()
		default:
			return
		}
	}
}

// 添加自动扩缩容方法
func (p *ConnectionPool) autoScale() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.RLock()
		currentSize := atomic.LoadInt32(&p.currentSize)
		queueSize := len(p.connections)
		capacity := cap(p.connections)
		p.mu.RUnlock()

		// 计算使用率
		utilizationRate := float64(capacity-queueSize) / float64(capacity)

		// 扩容: 使用率超过80%且未达到最大值
		if utilizationRate > 0.8 && currentSize < int32(p.maxSize) {
			p.scale(true)
		}

		// 缩容: 使用率低于30%且超过最小值
		if utilizationRate < 0.3 && currentSize > int32(p.minSize) {
			p.scale(false)
		}
	}
}

// 添加扩缩容执行方法
func (p *ConnectionPool) scale(isExpand bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	currentSize := atomic.LoadInt32(&p.currentSize)

	if isExpand {
		// 扩容: 每次增加 25% 连接数
		expandSize := int(float64(currentSize) * 0.25)
		if expandSize < 1 {
			expandSize = 1
		}

		for i := 0; i < expandSize && currentSize < int32(p.maxSize); i++ {
			if conn, err := createNewConnection(p.addr); err == nil {
				select {
				case p.connections <- conn:
					atomic.AddInt32(&p.currentSize, 1)
				default:
					conn.Close()
				}
			}
		}
	} else {
		// 缩容: 每次减少 25% 连接数
		shrinkSize := int(float64(currentSize) * 0.25)
		if shrinkSize < 1 {
			shrinkSize = 1
		}

		for i := 0; i < shrinkSize && currentSize > int32(p.minSize); i++ {
			select {
			case conn := <-p.connections:
				conn.Close()
				atomic.AddInt32(&p.currentSize, -1)
			default:
				return
			}
		}
	}
}
