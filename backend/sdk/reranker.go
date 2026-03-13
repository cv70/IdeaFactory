package sdk

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/cv70/pkgo/gslice"
	"github.com/cv70/pkgo/gtime"
)

type Reranker interface {
	Rerank(query string, items []string, config *RerankConfig) ([]*RerankItem, error)
}

var _ Reranker = (*AnythingRerankerClient)(nil)

type AnythingRerankerClient struct {
	Model string
	URL   string
}

type RerankItem struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
}

type RerankResp struct {
	Data []*RerankItem `json:"data"`
}

const (
	instruction = `Find relevant exam questions from the candidate set based on the query term`

	prefix = `<|im_start|>system\n请根据提供的查询（Query）和指令（Instruct），判断文档（Document）是否满足要求。答案只能是“是”或“否”。<|im_end|>\n<|im_start|>user\n`
	suffix = "<|im_end|>\n<|im_start|>assistant\n<think>\n\n</think>\n\n"

	queryTemplate    = `{prefix}<Instruct>: {instruction}\n<Query>: {query}\n`
	documentTemplate = `<Document>: {doc}{suffix}`
)

type RerankConfig struct {
	Instruction string
}

func (r *AnythingRerankerClient) Rerank(query string, items []string, config *RerankConfig) ([]*RerankItem, error) {
	defer gtime.LogTimeCost(0)()
	if config != nil && config.Instruction != "" {
		query = strings.ReplaceAll(queryTemplate, "{query}", query)
		query = strings.ReplaceAll(query, "{prefix}", prefix)
		query = strings.ReplaceAll(query, "{instruction}", config.Instruction)
	}
	items = gslice.Map(items, func(item string) string {
		item = strings.ReplaceAll(documentTemplate, "{item}", item)
		item = strings.ReplaceAll(item, "{suffix}", suffix)
		return item
	})
	// req := map[string]any{
	// 	"query": query,
	// 	"documents": items,
	// 	"model": r.Model,
	// }
	req := map[string]any{
		"text_1":                 []string{query},
		"text_2":                 items,
		"truncate_prompt_tokens": -1,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	reqReader := bytes.NewBuffer(reqData)
	resp, err := http.Post(r.URL, "application/json", reqReader)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var rerankResp RerankResp
	err = json.Unmarshal(respData, &rerankResp)
	return rerankResp.Data, err
}
