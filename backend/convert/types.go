package convert

import (
	"net/http"
	"sync"

	"github.com/sbgayhub/chameleon/backend/channel"
)

// Converter 转换器接口
type Converter interface {
	Name() string                                                                             // 转换器名称，ex: "openai->anthropic"
	ConvertRequest(request *http.Request, channel channel.Channel) (*http.Request, error)     // 转换请求数据
	ConvertResponse(response *http.Response, channel channel.Channel) (*http.Response, error) // 转换响应数据，返回转换后的数据和 token 使用信息
	ConvertStream(response *http.Response, channel channel.Channel) (*http.Response, error)   // 转换响应数据，返回转换后的数据和 token 使用信息
}

// TokenUsage token 使用信息
type TokenUsage struct {
	InputTokens  uint64
	OutputTokens uint64
}

// Registry 转换器注册表
type Registry struct {
	converters map[string]Converter
	mu         sync.RWMutex
}

// OpenAIRequest OpenAI 请求结构 (根据 newapi.ai 文档)
type OpenAIRequest struct {
	Model               string          `json:"model"`
	Messages            []OpenAIMessage `json:"messages"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"` // 替代已弃用的 max_tokens
	MaxTokens           int             `json:"max_tokens,omitempty"`            // 已弃用，保留兼容性
	Temperature         float64         `json:"temperature,omitempty"`
	TopP                float64         `json:"top_p,omitempty"`
	Stream              bool            `json:"stream,omitempty"`
	// 推理相关参数 (o1 系列)
	ReasoningEffort string `json:"reasoning_effort,omitempty"` // "low", "medium", "high"
	// 其他参数
	Stop             []string `json:"stop,omitempty"`
	PresencePenalty  float64  `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64  `json:"frequency_penalty,omitempty"`
	User             string   `json:"user,omitempty"`
}

type OpenAIMessage struct {
	Role    string      `json:"role"`           // "system", "user", "assistant", "tool", "developer"
	Content interface{} `json:"content"`        // string 或 content array
	Name    string      `json:"name,omitempty"` // 可选的参与者名称
	// 工具调用相关 (暂不完整实现)
	ToolCalls  []interface{} `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

// OpenAIResponse OpenAI 响应结构 (根据 newapi.ai 文档)
type OpenAIResponse struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"` // "chat.completion"
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	SystemFingerprint string         `json:"system_fingerprint,omitempty"`
	Choices           []OpenAIChoice `json:"choices"`
	Usage             OpenAIUsage    `json:"usage"`
	ServiceTier       string         `json:"service_tier,omitempty"`
}

type OpenAIChoice struct {
	Index        int                   `json:"index"`
	Message      OpenAIMessageInChoice `json:"message"`
	Logprobs     *OpenAILogprobs       `json:"logprobs,omitempty"`
	FinishReason string                `json:"finish_reason"` // "stop", "length", "tool_calls", "content_filter"
}

type OpenAIMessageInChoice struct {
	Role             string        `json:"role"`                        // "assistant"
	Content          interface{}   `json:"content"`                     // string 或 null
	Refusal          interface{}   `json:"refusal,omitempty"`           // 拒绝消息
	ReasoningContent string        `json:"reasoning_content,omitempty"` // 推理内容 (o1 模型)
	Annotations      []interface{} `json:"annotations,omitempty"`       // 注释
	ToolCalls        []interface{} `json:"tool_calls,omitempty"`        // 工具调用
}

type OpenAILogprobs struct {
	Content []OpenAITokenLogprob `json:"content,omitempty"`
	Refusal []OpenAITokenLogprob `json:"refusal,omitempty"`
}

type OpenAITokenLogprob struct {
	Token       string             `json:"token"`
	Logprob     float64            `json:"logprob"`
	Bytes       []int              `json:"bytes,omitempty"`
	TopLogprobs []OpenAITopLogprob `json:"top_logprobs,omitempty"`
}

type OpenAITopLogprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

type OpenAIUsage struct {
	PromptTokens            int                            `json:"prompt_tokens"`
	CompletionTokens        int                            `json:"completion_tokens"`
	TotalTokens             int                            `json:"total_tokens"`
	PromptTokensDetails     *OpenAIPromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *OpenAICompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

type OpenAIPromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
	AudioTokens  int `json:"audio_tokens,omitempty"`
}

type OpenAICompletionTokensDetails struct {
	ReasoningTokens          int `json:"reasoning_tokens,omitempty"`
	AudioTokens              int `json:"audio_tokens,omitempty"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty"`
}

// OpenAIStreamResponse OpenAI 流式响应结构 (根据 newapi.ai 文档)
type OpenAIStreamResponse struct {
	ID                string               `json:"id"`
	Object            string               `json:"object"` // "chat.completion.chunk"
	Created           int64                `json:"created"`
	Model             string               `json:"model"`
	SystemFingerprint string               `json:"system_fingerprint,omitempty"`
	Choices           []OpenAIStreamChoice `json:"choices"`
	Usage             OpenAIUsage          `json:"usage,omitempty"`
}

type OpenAIStreamChoice struct {
	Index        int             `json:"index"`
	Delta        OpenAIDelta     `json:"delta"`
	Logprobs     *OpenAILogprobs `json:"logprobs,omitempty"`
	FinishReason *string         `json:"finish_reason,omitempty"`
}

type OpenAIDelta struct {
	Role      string        `json:"role,omitempty"`       // "assistant" 在第一个块中
	Content   string        `json:"content,omitempty"`    // 增量内容
	Thinking  string        `json:"thinking,omitempty"`   // 思考内容
	ToolCalls []interface{} `json:"tool_calls,omitempty"` // 工具调用
}

// AnthropicRequest Anthropic 请求结构 (根据 newapi.ai 文档)
type AnthropicRequest struct {
	Model         string             `json:"model"`
	MaxTokens     int                `json:"max_tokens"` // 必需
	Temperature   float64            `json:"temperature,omitempty"`
	TopP          float64            `json:"top_p,omitempty"`
	TopK          int                `json:"top_k,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	Messages      []AnthropicMessage `json:"messages"` // 必需
	// Anthropic 特有参数
	System     string               `json:"system,omitempty"`      // 系统提示词
	Metadata   *AnthropicMetadata   `json:"metadata,omitempty"`    // 元数据
	Thinking   *AnthropicThinking   `json:"thinking,omitempty"`    // 思考功能
	ToolChoice *AnthropicToolChoice `json:"tool_choice,omitempty"` // 工具选择
	Tools      []AnthropicTool      `json:"tools,omitempty"`       // 工具定义
}

type AnthropicMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

type AnthropicThinking struct {
	Type         string `json:"type"`                    // "enabled" 或 "disabled"
	BudgetTokens int    `json:"budget_tokens,omitempty"` // 当 type="enabled" 时必需，>= 1024
}

type AnthropicToolChoice struct {
	Type                   string `json:"type"`           // "auto", "any", "tool"
	Name                   string `json:"name,omitempty"` // 当 type="tool" 时必需
	DisableParallelToolUse bool   `json:"disable_parallel_tool_use,omitempty"`
}

type AnthropicTool struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	InputSchema  map[string]interface{} `json:"input_schema"` // JSON Schema
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

type AnthropicCacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

type AnthropicMessage struct {
	Role    string      `json:"role"`    // "user", "assistant"
	Content interface{} `json:"content"` // string 或 []AnthropicContent
}

// AnthropicContent 支持的内容类型
type AnthropicContent struct {
	Type         string                 `json:"type"` // "text", "image", "tool_use", "tool_result", "document"
	Text         string                 `json:"text"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`

	// Image 相关
	Source *AnthropicSource `json:"source,omitempty"`

	// Tool Use 相关
	ID    string                 `json:"id,omitempty"`    // tool_use 的 ID
	Name  string                 `json:"name,omitempty"`  // tool_use 的名称
	Input map[string]interface{} `json:"input,omitempty"` // tool_use 的输入参数

	// Tool Result 相关
	ToolUseID string `json:"tool_use_id,omitempty"` // tool_result 的 ID
	IsError   bool   `json:"is_error,omitempty"`    // tool_result 是否错误

}

type AnthropicSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/jpeg", "image/png", "image/gif", "image/webp", "application/pdf" 等
	Data      string `json:"data"`       // base64 编码的图片数据
}

// AnthropicResponse Anthropic 响应结构 (根据 newapi.ai 文档)
type AnthropicResponse struct {
	ID              string             `json:"id"`
	Type            string             `json:"type"`                       // "message"
	Content         []AnthropicContent `json:"content"`                    // 响应内容
	ThinkingContent []AnthropicContent `json:"thinking_content,omitempty"` // 思考内容 (Claude 3.7+)
	Model           string             `json:"model"`
	Role            string             `json:"role"`        // "assistant"
	StopReason      string             `json:"stop_reason"` // "end_turn", "max_tokens", "stop_sequence", "tool_use"
	StopSequence    *string            `json:"stop_sequence,omitempty"`
	Usage           AnthropicUsage     `json:"usage"`
}

type AnthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	ThinkingTokens           int `json:"thinking_tokens,omitempty"`             // 思考token (Claude 3.7+)
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"` // 缓存创建token
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`     // 缓存读取token
}

// AnthropicStreamResponse Anthropic 流式响应结构 (根据 newapi.ai 文档)
type AnthropicStreamResponse struct {
	Type         string                  `json:"type"`  // 事件类型：message_start | content_block_start | content_block_stop | message_stop
	Index        int                     `json:"index"` // content_block_delta 事件必需，不能省略
	ContentBlock *AnthropicContent       `json:"content_block,omitempty"`
	Delta        *AnthropicDelta         `json:"delta,omitempty"`
	Message      *AnthropicStreamMessage `json:"message,omitempty"`
	Usage        *AnthropicUsage         `json:"usage,omitempty"`
}

type AnthropicStreamMessage struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"` // "message"
	Role    string             `json:"role"` // "assistant"
	Model   string             `json:"model"`
	Usage   *AnthropicUsage    `json:"usage,omitempty"`
	Content []AnthropicContent `json:"content"`
}

type AnthropicDelta struct {
	Type       string `json:"type"` // text_delta | thinking_delta
	Text       string `json:"text"` // text_delta 事件必需，可以为空字符串但不能省略
	Thinking   string `json:"thinking,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}

// ========== GEMINI 数据结构 ==========

// GeminiRequest Gemini 请求结构 (根据 newapi.ai 文档)
type GeminiRequest struct {
	Contents          []GeminiContent         `json:"contents"` // 必需
	GenerationConfig  *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	SystemInstruction *GeminiContent          `json:"systemInstruction,omitempty"`
	Tools             []GeminiTool            `json:"tools,omitempty"`
	ToolConfig        *GeminiToolConfig       `json:"toolConfig,omitempty"`
	SafetySettings    []GeminiSafetySetting   `json:"safetySettings,omitempty"`
	CachedContent     string                  `json:"cachedContent,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"` // "user", "model", "function", "tool"
	Parts []GeminiPart `json:"parts"`          // 必需
}

type GeminiPart struct {
	// 文本内容
	Text string `json:"text,omitempty"`

	// 内联媒体数据 (图片、音频、视频、PDF)
	InlineData *GeminiInlineData `json:"inline_data,omitempty"`

	// 文件数据
	FileData *GeminiFileData `json:"file_data,omitempty"`

	// 函数调用
	FunctionCall *GeminiFunctionCall `json:"functionCall,omitempty"`

	// 函数响应
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`

	// 可执行代码
	ExecutableCode *GeminiExecutableCode `json:"executableCode,omitempty"`

	// 代码执行结果
	CodeExecutionResult *GeminiCodeExecutionResult `json:"codeExecutionResult,omitempty"`
}

type GeminiInlineData struct {
	MimeType string `json:"mimeType"` // 媒体类型
	Data     string `json:"data"`     // base64 编码的数据
}

type GeminiFileData struct {
	MimeType string `json:"mimeType"` // 文件类型
	FileURI  string `json:"file_uri"` // 文件URI
}

type GeminiFunctionCall struct {
	Name string                 `json:"name"` // 函数名
	Args map[string]interface{} `json:"args"` // 参数
}

type GeminiFunctionResponse struct {
	Name     string                 `json:"name"`     // 函数名
	Response map[string]interface{} `json:"response"` // 响应数据
}

type GeminiExecutableCode struct {
	Language string `json:"language"` // 编程语言，如 "PYTHON"
	Code     string `json:"code"`     // 要执行的代码
}

type GeminiCodeExecutionResult struct {
	Outcome string `json:"outcome"`          // "OUTCOME_OK", "OUTCOME_FAILED", "OUTCOME_DEADLINE_EXCEEDED"
	Output  string `json:"output,omitempty"` // 输出内容
}

type GeminiGenerationConfig struct {
	Temperature                float64               `json:"temperature,omitempty"`
	TopP                       float64               `json:"topP,omitempty"`
	TopK                       int                   `json:"topK,omitempty"`
	MaxOutputTokens            int                   `json:"maxOutputTokens,omitempty"`
	StopSequences              []string              `json:"stopSequences,omitempty"`
	ResponseMIMEType           string                `json:"responseMimeType,omitempty"` // "text/plain", "application/json"
	ResponseSchema             *GeminiJSONSchema     `json:"responseSchema,omitempty"`   // JSON Schema
	CandidateCount             int                   `json:"candidateCount,omitempty"`
	PresencePenalty            float64               `json:"presencePenalty,omitempty"`
	FrequencyPenalty           float64               `json:"frequencyPenalty,omitempty"`
	Seed                       int                   `json:"seed,omitempty"`
	ResponseModalities         []string              `json:"responseModalities,omitempty"` // "TEXT", "IMAGE", "AUDIO"
	MediaResolution            string                `json:"mediaResolution,omitempty"`    // "MEDIA_RESOLUTION_LOW", "MEDIUM", "HIGH"
	ThinkingConfig             *GeminiThinkingConfig `json:"thinkingConfig,omitempty"`     // 思考配置
	SpeechConfig               *GeminiSpeechConfig   `json:"speechConfig,omitempty"`       // 语音配置
	EnableEnhancedCivicAnswers bool                  `json:"enableEnhancedCivicAnswers,omitempty"`
}

type GeminiJSONSchema struct {
	Type        string                       `json:"type"`
	Description string                       `json:"description,omitempty"`
	Enum        []string                     `json:"enum,omitempty"`
	Example     interface{}                  `json:"example,omitempty"`
	Nullable    bool                         `json:"nullable,omitempty"`
	Format      string                       `json:"format,omitempty"`
	Items       *GeminiJSONSchema            `json:"items,omitempty"`
	Properties  map[string]*GeminiJSONSchema `json:"properties,omitempty"`
	Required    []string                     `json:"required,omitempty"`
	Minimum     *float64                     `json:"minimum,omitempty"`
	Maximum     *float64                     `json:"maximum,omitempty"`
	MinItems    *int                         `json:"minItems,omitempty"`
	MaxItems    *int                         `json:"maxItems,omitempty"`
	MinLength   *int                         `json:"minLength,omitempty"`
	MaxLength   *int                         `json:"maxLength,omitempty"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
	CodeExecution        *GeminiCodeExecution        `json:"codeExecution,omitempty"`
}

type GeminiCodeExecution struct {
	// 空对象，表示启用代码执行
}

type GeminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type GeminiToolConfig struct {
	FunctionCallingConfig *GeminiFunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

type GeminiFunctionCallingConfig struct {
	Mode                 string   `json:"mode,omitempty"`                 // "MODE_UNSPECIFIED", "AUTO", "ANY", "NONE"
	AllowedFunctionNames []string `json:"allowedFunctionNames,omitempty"` // 允许调用的函数名
}

type GeminiThinkingConfig struct {
	IncludeThoughts bool `json:"includeThoughts,omitempty"` // 是否在回答中包含思考内容
	ThinkingBudget  int  `json:"thinkingBudget,omitempty"`  // 模型应生成的想法token的数量
}

type GeminiSpeechConfig struct {
	VoiceConfig             *GeminiVoiceConfig             `json:"voiceConfig,omitempty"`
	MultiSpeakerVoiceConfig *GeminiMultiSpeakerVoiceConfig `json:"multiSpeakerVoiceConfig,omitempty"`
	LanguageCode            string                         `json:"languageCode,omitempty"`
}

type GeminiVoiceConfig struct {
	PrebuiltVoiceConfig *GeminiPrebuiltVoiceConfig `json:"prebuiltVoiceConfig,omitempty"`
}

type GeminiPrebuiltVoiceConfig struct {
	VoiceName string `json:"voiceName"`
}

type GeminiMultiSpeakerVoiceConfig struct {
	SpeakerVoiceConfigs []GeminiSpeakerVoiceConfig `json:"speakerVoiceConfigs"`
}

type GeminiSpeakerVoiceConfig struct {
	Speaker     string             `json:"speaker"`
	VoiceConfig *GeminiVoiceConfig `json:"voiceConfig"`
}

type GeminiSafetySetting struct {
	Category  string `json:"category"`  // "HARM_CATEGORY_HARASSMENT", "HARM_CATEGORY_HATE_SPEECH", etc.
	Threshold string `json:"threshold"` // "BLOCK_LOW_AND_ABOVE", "BLOCK_MEDIUM_AND_ABOVE", "BLOCK_ONLY_HIGH", "BLOCK_NONE", "OFF"
}

// GeminiResponse Gemini 响应结构 (根据 newapi.ai 文档)
type GeminiResponse struct {
	Candidates     []GeminiCandidate     `json:"candidates"`
	PromptFeedback *GeminiPromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *GeminiUsageMetadata  `json:"usageMetadata,omitempty"`
	ModelVersion   string                `json:"modelVersion,omitempty"`
	ResponseID     string                `json:"responseId,omitempty"`
}

type GeminiCandidate struct {
	Content               GeminiContent                `json:"content"`
	FinishReason          string                       `json:"finishReason"` // "STOP", "MAX_TOKENS", "SAFETY", "RECITATION", "OTHER"
	Index                 int                          `json:"index"`
	SafetyRatings         []GeminiSafetyRating         `json:"safetyRatings,omitempty"`
	CitationMetadata      *GeminiCitationMetadata      `json:"citationMetadata,omitempty"`
	TokenCount            int                          `json:"tokenCount,omitempty"`
	GroundingAttributions []GeminiGroundingAttribution `json:"groundingAttributions,omitempty"`
	GroundingMetadata     *GeminiGroundingMetadata     `json:"groundingMetadata,omitempty"`
	AvgLogprobs           float64                      `json:"avgLogprobs,omitempty"`
	LogprobsResult        *GeminiLogprobsResult        `json:"logprobsResult,omitempty"`
	URLRetrievalMetadata  *GeminiURLRetrievalMetadata  `json:"urlRetrievalMetadata,omitempty"`
	URLContextMetadata    *GeminiURLContextMetadata    `json:"urlContextMetadata,omitempty"`
}

type GeminiSafetyRating struct {
	Category    string `json:"category"`    // "HARM_CATEGORY_HARASSMENT", "HARM_CATEGORY_HATE_SPEECH", etc.
	Probability string `json:"probability"` // "NEGLIGIBLE", "LOW", "MEDIUM", "HIGH"
	Blocked     bool   `json:"blocked"`
}

type GeminiCitationMetadata struct {
	CitationSources []GeminiCitationSource `json:"citationSources,omitempty"`
}

type GeminiCitationSource struct {
	StartIndex int    `json:"startIndex"`
	EndIndex   int    `json:"endIndex"`
	URI        string `json:"uri,omitempty"`
	License    string `json:"license,omitempty"`
}

type GeminiGroundingAttribution struct {
	SourceID *GeminiAttributionSourceID `json:"sourceId,omitempty"`
	Content  *GeminiAttributionSource   `json:"content,omitempty"`
}

type GeminiAttributionSourceID struct {
	GroundingPassage       *GeminiGroundingPassageID     `json:"groundingPassage,omitempty"`
	SemanticRetrieverChunk *GeminiSemanticRetrieverChunk `json:"semanticRetrieverChunk,omitempty"`
}

type GeminiGroundingPassageID struct {
	PassageID string `json:"passageId"`
	PartIndex int    `json:"partIndex"`
}

type GeminiSemanticRetrieverChunk struct {
	Source string `json:"source"`
	Chunk  string `json:"chunk"`
}

type GeminiAttributionSource struct {
	GroundingPassage       *GeminiGroundingPassageID     `json:"groundingPassage,omitempty"`
	SemanticRetrieverChunk *GeminiSemanticRetrieverChunk `json:"semanticRetrieverChunk,omitempty"`
}

type GeminiGroundingMetadata struct {
	GroundingChunks   []GeminiGroundingChunk   `json:"groundingChunks,omitempty"`
	GroundingSupports []GeminiGroundingSupport `json:"groundingSupports,omitempty"`
	WebSearchQueries  []string                 `json:"webSearchQueries,omitempty"`
	SearchEntryPoint  *GeminiSearchEntryPoint  `json:"searchEntryPoint,omitempty"`
	RetrievalMetadata *GeminiRetrievalMetadata `json:"retrievalMetadata,omitempty"`
}

type GeminiGroundingChunk struct {
	Web *GeminiWeb `json:"web,omitempty"`
}

type GeminiWeb struct {
	URI   string `json:"uri,omitempty"`
	Title string `json:"title,omitempty"`
}

type GeminiGroundingSupport struct {
	GroundingChunkIndices []int          `json:"groundingChunkIndices,omitempty"`
	ConfidenceScores      []float64      `json:"confidenceScores,omitempty"`
	Segment               *GeminiSegment `json:"segment,omitempty"`
}

type GeminiSegment struct {
	PartIndex  int    `json:"partIndex,omitempty"`
	StartIndex int    `json:"startIndex,omitempty"`
	EndIndex   int    `json:"endIndex,omitempty"`
	Text       string `json:"text,omitempty"`
}

type GeminiSearchEntryPoint struct {
	RenderedContent string `json:"renderedContent,omitempty"`
	SDKBlob         string `json:"sdkBlob,omitempty"`
}

type GeminiRetrievalMetadata struct {
	GoogleSearchDynamicRetrievalScore float64 `json:"googleSearchDynamicRetrievalScore,omitempty"`
}

type GeminiLogprobsResult struct {
	TopCandidates    []GeminiTopCandidates    `json:"topCandidates,omitempty"`
	ChosenCandidates []GeminiChosenCandidates `json:"chosenCandidates,omitempty"`
}

type GeminiTopCandidates struct {
	Candidates []GeminiCandidateLogprob `json:"candidates"`
}

type GeminiChosenCandidates struct {
	Candidates []GeminiCandidateLogprob `json:"candidates"`
}

type GeminiCandidateLogprob struct {
	Token          string  `json:"token"`
	TokenID        int     `json:"tokenId,omitempty"`
	LogProbability float64 `json:"logProbability"`
}

type GeminiURLRetrievalMetadata struct {
	URLRetrievalContexts []GeminiURLRetrievalContext `json:"urlRetrievalContexts,omitempty"`
}

type GeminiURLRetrievalContext struct {
	RetrievedURL string `json:"retrievedUrl"`
}

type GeminiURLContextMetadata struct {
	URLMetadata []GeminiURLMetadata `json:"urlMetadata,omitempty"`
}

type GeminiURLMetadata struct {
	RetrievedURL       string `json:"retrievedUrl"`
	URLRetrievalStatus string `json:"urlRetrievalStatus"` // "URL_RETRIEVAL_STATUS_SUCCESS", "URL_RETRIEVAL_STATUS_ERROR"
}

type GeminiPromptFeedback struct {
	BlockReason   string               `json:"blockReason,omitempty"` // "SAFETY", "OTHER", "BLOCKLIST", "PROHIBITED_CONTENT", "IMAGE_SAFETY"
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings,omitempty"`
}

type GeminiUsageMetadata struct {
	PromptTokenCount           int                        `json:"promptTokenCount"`
	CachedContentTokenCount    int                        `json:"cachedContentTokenCount,omitempty"`
	CandidatesTokenCount       int                        `json:"candidatesTokenCount"`
	TotalTokenCount            int                        `json:"totalTokenCount"`
	ToolUsePromptTokenCount    int                        `json:"toolUsePromptTokenCount,omitempty"`
	ThoughtsTokenCount         int                        `json:"thoughtsTokenCount,omitempty"`
	PromptTokensDetails        []GeminiModalityTokenCount `json:"promptTokensDetails,omitempty"`
	CandidatesTokensDetails    []GeminiModalityTokenCount `json:"candidatesTokensDetails,omitempty"`
	CacheTokensDetails         []GeminiModalityTokenCount `json:"cacheTokensDetails,omitempty"`
	ToolUsePromptTokensDetails []GeminiModalityTokenCount `json:"toolUsePromptTokensDetails,omitempty"`
}

type GeminiModalityTokenCount struct {
	Modality   string `json:"modality"` // "TEXT", "IMAGE", "AUDIO", "VIDEO", "DOCUMENT"
	TokenCount int    `json:"tokenCount"`
}

// GeminiStreamResponse Gemini 流式响应结构
type GeminiStreamResponse struct {
	Candidates     []GeminiCandidate     `json:"candidates"`
	PromptFeedback *GeminiPromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *GeminiUsageMetadata  `json:"usageMetadata,omitempty"`
	ModelVersion   string                `json:"modelVersion,omitempty"`
	ResponseID     string                `json:"responseId,omitempty"`
}
