package llmproxy

import (
	"context"

	"github.com/earnlearning/backend/internal/application"
)

// ChatAdapter implements application.ChatLLMClient, forwarding to *Client.
type ChatAdapter struct{ c *Client }

func NewChatAdapter(c *Client) *ChatAdapter { return &ChatAdapter{c: c} }

// Stats — application.ChatLLMClient.Stats 구현 (#088 큐잉 진행률용).
func (a *ChatAdapter) Stats() application.LLMStats {
	s := a.c.ChatStats()
	return application.LLMStats{
		InFlight: s.InFlight,
		Waiting:  s.Waiting,
		Cap:      s.Cap,
	}
}

func (a *ChatAdapter) ChatComplete(ctx context.Context, req *application.LLMChatRequest) (*application.LLMChatResponse, error) {
	if req == nil {
		return nil, nil
	}
	// translate
	msgs := make([]ChatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		tcs := make([]ChatToolCall, 0, len(m.ToolCalls))
		for _, tc := range m.ToolCalls {
			tcs = append(tcs, ChatToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: ChatToolFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		// #106 vision: ContentParts 가 비어있지 않으면 multimodal 메시지로 전송
		var parts []ContentBlock
		if len(m.ContentParts) > 0 {
			parts = make([]ContentBlock, 0, len(m.ContentParts))
			for _, p := range m.ContentParts {
				cb := ContentBlock{Type: p.Type, Text: p.Text}
				if p.ImageURL != "" {
					cb.ImageURL = &ContentImage{URL: p.ImageURL}
				}
				parts = append(parts, cb)
			}
		}
		msgs = append(msgs, ChatMessage{
			Role:         m.Role,
			Content:      m.Content,
			ContentParts: parts,
			ToolCalls:    tcs,
			ToolCallID:   m.ToolCallID,
			Name:         m.Name,
		})
	}
	tools := make([]ChatToolSpec, 0, len(req.Tools))
	for _, t := range req.Tools {
		tools = append(tools, ChatToolSpec{
			Type: t.Type,
			Function: ChatToolSpecFunction{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		})
	}
	inner := &ChatRequest{
		Model:           req.Model,
		Messages:        msgs,
		MaxTokens:       req.MaxTokens,
		ReasoningEffort: req.ReasoningEffort,
		Tools:           tools,
		ToolChoice:      req.ToolChoice,
	}
	resp, err := a.c.ChatComplete(ctx, inner)
	if err != nil {
		return nil, err
	}
	out := &application.LLMChatResponse{
		Model: resp.Model,
		Usage: application.LLMChatUsage{
			PromptTokens:       resp.Usage.PromptTokens,
			CompletionTokens:   resp.Usage.CompletionTokens,
			PromptCachedTokens: resp.Usage.PromptCachedTokens,
		},
	}
	for _, ch := range resp.Choices {
		outMsgToolCalls := make([]application.LLMChatToolCall, 0, len(ch.Message.ToolCalls))
		for _, tc := range ch.Message.ToolCalls {
			outMsgToolCalls = append(outMsgToolCalls, application.LLMChatToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: application.LLMChatToolFunc{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		out.Choices = append(out.Choices, application.LLMChatChoice{
			Message: application.LLMChatMessage{
				Role:       ch.Message.Role,
				Content:    ch.Message.Content,
				ToolCalls:  outMsgToolCalls,
				ToolCallID: ch.Message.ToolCallID,
				Name:       ch.Message.Name,
			},
			FinishReason: ch.FinishReason,
		})
	}
	return out, nil
}

// ChatCompleteStream — application.ChatLLMClient 의 streaming 메서드.
// 내부 ChatStreamEvent 를 application.LLMStreamEvent 로 변환.
func (a *ChatAdapter) ChatCompleteStream(ctx context.Context, req *application.LLMChatRequest) (<-chan application.LLMStreamEvent, error) {
	if req == nil {
		return nil, nil
	}
	msgs := make([]ChatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		tcs := make([]ChatToolCall, 0, len(m.ToolCalls))
		for _, tc := range m.ToolCalls {
			tcs = append(tcs, ChatToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: ChatToolFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		// #106 vision: ContentParts 가 비어있지 않으면 multimodal 메시지로 전송
		var parts []ContentBlock
		if len(m.ContentParts) > 0 {
			parts = make([]ContentBlock, 0, len(m.ContentParts))
			for _, p := range m.ContentParts {
				cb := ContentBlock{Type: p.Type, Text: p.Text}
				if p.ImageURL != "" {
					cb.ImageURL = &ContentImage{URL: p.ImageURL}
				}
				parts = append(parts, cb)
			}
		}
		msgs = append(msgs, ChatMessage{
			Role:         m.Role,
			Content:      m.Content,
			ContentParts: parts,
			ToolCalls:    tcs,
			ToolCallID:   m.ToolCallID,
			Name:         m.Name,
		})
	}
	tools := make([]ChatToolSpec, 0, len(req.Tools))
	for _, t := range req.Tools {
		tools = append(tools, ChatToolSpec{
			Type: t.Type,
			Function: ChatToolSpecFunction{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		})
	}
	inner := &ChatRequest{
		Model:           req.Model,
		Messages:        msgs,
		MaxTokens:       req.MaxTokens,
		ReasoningEffort: req.ReasoningEffort,
		Tools:           tools,
		ToolChoice:      req.ToolChoice,
	}
	src, err := a.c.ChatCompleteStream(ctx, inner)
	if err != nil {
		return nil, err
	}
	out := make(chan application.LLMStreamEvent, 16)
	go func() {
		defer close(out)
		for ev := range src {
			outEv := application.LLMStreamEvent{
				TextDelta:    ev.TextDelta,
				FinishReason: ev.FinishReason,
				Err:          ev.Err,
			}
			if ev.Usage != nil {
				outEv.Usage = &application.LLMChatUsage{
					PromptTokens:       ev.Usage.PromptTokens,
					CompletionTokens:   ev.Usage.CompletionTokens,
					PromptCachedTokens: ev.Usage.PromptCachedTokens,
				}
			}
			out <- outEv
		}
	}()
	return out, nil
}
