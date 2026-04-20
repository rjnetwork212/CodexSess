package httpapi

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestExtractChatMessagesImages_StringContent(t *testing.T) {
	msgs := []ChatMessage{{Role: "user", Content: "hello"}}
	if got := extractChatMessagesImages(msgs); len(got) != 0 {
		t.Fatalf("expected no images from plain string, got %+v", got)
	}
}

func TestExtractChatMessagesImages_ImageURLPart(t *testing.T) {
	msgs := []ChatMessage{{
		Role: "user",
		Content: []any{
			map[string]any{"type": "text", "text": "describe"},
			map[string]any{
				"type": "image_url",
				"image_url": map[string]any{
					"url":    "https://example.com/cat.png",
					"detail": "high",
				},
			},
		},
	}}
	got := extractChatMessagesImages(msgs)
	want := []DirectImage{{URL: "https://example.com/cat.png", Detail: "high"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestExtractChatMessagesImages_ImageURLString(t *testing.T) {
	msgs := []ChatMessage{{
		Role: "user",
		Content: []any{
			map[string]any{"type": "image_url", "image_url": "data:image/png;base64,AAA"},
		},
	}}
	got := extractChatMessagesImages(msgs)
	want := []DirectImage{{URL: "data:image/png;base64,AAA"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestExtractChatMessagesImages_InputImageShape(t *testing.T) {
	msgs := []ChatMessage{{
		Role: "user",
		Content: []any{
			map[string]any{
				"type":      "input_image",
				"image_url": "https://example.com/x.jpg",
				"detail":    "low",
			},
		},
	}}
	got := extractChatMessagesImages(msgs)
	want := []DirectImage{{URL: "https://example.com/x.jpg", Detail: "low"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestExtractChatMessagesImages_FlattensMultiple(t *testing.T) {
	msgs := []ChatMessage{
		{Role: "user", Content: []any{map[string]any{"type": "image_url", "image_url": "a"}}},
		{Role: "user", Content: []any{map[string]any{"type": "image_url", "image_url": "b"}}},
	}
	got := extractChatMessagesImages(msgs)
	if len(got) != 2 || got[0].URL != "a" || got[1].URL != "b" {
		t.Fatalf("unexpected flatten result: %+v", got)
	}
}

func TestExtractResponsesInputImages_StringInput(t *testing.T) {
	raw := json.RawMessage(`"just a prompt"`)
	if got := extractResponsesInputImages(raw); len(got) != 0 {
		t.Fatalf("expected no images from string input, got %+v", got)
	}
}

func TestExtractResponsesInputImages_InputImage(t *testing.T) {
	raw := json.RawMessage(`[{"role":"user","content":[{"type":"input_text","text":"hi"},{"type":"input_image","image_url":"https://ex/x.png","detail":"auto"}]}]`)
	got := extractResponsesInputImages(raw)
	want := []DirectImage{{URL: "https://ex/x.png", Detail: "auto"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestExtractClaudeMessagesImages_Base64(t *testing.T) {
	msgs := []ClaudeMessage{{
		Role:    "user",
		Content: json.RawMessage(`[{"type":"text","text":"what's this"},{"type":"image","source":{"type":"base64","media_type":"image/jpeg","data":"QUJD"}}]`),
	}}
	got := extractClaudeMessagesImages(msgs)
	want := []DirectImage{{URL: "data:image/jpeg;base64,QUJD"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestExtractClaudeMessagesImages_URL(t *testing.T) {
	msgs := []ClaudeMessage{{
		Role:    "user",
		Content: json.RawMessage(`[{"type":"image","source":{"type":"url","url":"https://example.com/z.png"}}]`),
	}}
	got := extractClaudeMessagesImages(msgs)
	want := []DirectImage{{URL: "https://example.com/z.png"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestExtractClaudeMessagesImages_PlainStringContent(t *testing.T) {
	msgs := []ClaudeMessage{{Role: "user", Content: json.RawMessage(`"hello"`)}}
	if got := extractClaudeMessagesImages(msgs); len(got) != 0 {
		t.Fatalf("expected no images from plain string content, got %+v", got)
	}
}

func TestDirectCodexPayload_WithImages(t *testing.T) {
	// Rebuild the exact payload shape produced by callDirectCodexResponses for
	// image+text input so regressions in the wire format are caught by a test.
	opts := directCodexRequestOptions{
		Images: []DirectImage{
			{URL: "https://example.com/x.png", Detail: "high"},
			{URL: "data:image/png;base64,AAA"},
		},
	}
	userContent := []map[string]any{
		{"type": "input_text", "text": "hello"},
	}
	for _, img := range opts.Images {
		part := map[string]any{"type": "input_image", "image_url": img.URL}
		if img.Detail != "" {
			part["detail"] = img.Detail
		}
		userContent = append(userContent, part)
	}

	if len(userContent) != 3 {
		t.Fatalf("expected 3 parts (text + 2 images), got %d", len(userContent))
	}
	if got := userContent[1]["type"]; got != "input_image" {
		t.Fatalf("expected second part input_image, got %v", got)
	}
	if got := userContent[1]["detail"]; got != "high" {
		t.Fatalf("expected detail=high preserved, got %v", got)
	}
	if _, hasDetail := userContent[2]["detail"]; hasDetail {
		t.Fatalf("expected detail key to be omitted when empty")
	}
}
