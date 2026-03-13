package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

type TextEmbeder[F float32 | float64] interface {
	Embedding(ctx context.Context, ss ...string) ([][]F, error)
}

var _ TextEmbeder[float32] = &TextEmbeddingClient[float32]{}
var _ TextEmbeder[float64] = &TextEmbeddingClient[float64]{}

type TextEmbeddingClient[F float32 | float64] struct {
	URL string
}

type TextEmbeddingResult[F float32 | float64] struct {
	Data []struct{
		Index int `json:"index"`
		Embedding []F `json:"embedding"`
	} `json:"data"`
}

func (c *TextEmbeddingClient[F]) Embedding(ctx context.Context, ss ...string) ([][]F, error) {
	if len(ss) == 0 {
		return [][]F{}, nil
	}
	req := map[string]any{
		"input": ss,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	reqReader := bytes.NewBuffer(reqData)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, reqReader)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	embeddingResult := TextEmbeddingResult[F]{}
	err = json.Unmarshal(respData, &embeddingResult)
	if err != nil {
		return nil, err
	}

	embeddings := make([][]F, len(ss))
	for _, item := range embeddingResult.Data {
		embeddings[item.Index] = item.Embedding
	}

	return embeddings, nil
}
