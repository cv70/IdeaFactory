package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
)

type ImageEmbeder[F float32 | float64] interface {
	Embedding(ctx context.Context, ss ...io.Reader) ([][]F, error)
}

var _ ImageEmbeder[float32] = &ImageEmbeddingClient[float32]{}
var _ ImageEmbeder[float64] = &ImageEmbeddingClient[float64]{}

type ImageEmbeddingClient[F float32 | float64] struct {
	URL string
}

type ImageEmbeddingResult[F float32 | float64] struct {
	Results struct{
		Data map[string][]F `json:"data"`
	} `json:"results"`
}

func (c *ImageEmbeddingClient[F]) Embedding(ctx context.Context, ss ...io.Reader) ([][]F, error) {
	if len(ss) == 0 {
		return [][]F{}, nil
	}
	body, contentType, err := func () (*bytes.Buffer, string, error) {
		// 创建 buffer 用于构建 multipart 表单
		body := &bytes.Buffer{}
	  	writer := multipart.NewWriter(body)
	    defer writer.Close()
	
		for i, s := range ss {
		    // 添加文件字段
		    part, err := writer.CreateFormFile(strconv.Itoa(i), fmt.Sprintf("%d.jpg", i))
		    if err != nil {
		        return nil, "", err
		    }
		
		    // 将文件内容写入 part
		    _, err = io.Copy(part, s)
		    if err != nil {
		        return nil, "", err
		    }
    	}
		return body, writer.FormDataContentType(), nil
	}()
    
   	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, body)
	if err != nil {
		return nil, err
	}

	// 设置 Content-Type（由 multipart writer 自动设置）
    req.Header.Set("Content-Type", contentType)

    resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	embeddingResult := ImageEmbeddingResult[F]{}
	err = json.Unmarshal(respData, &embeddingResult)
	if err != nil {
		return nil, err
	}

	index2Embedding := map[int][]F{}
	for i, embedding := range embeddingResult.Results.Data {
		idx, err := strconv.Atoi(i)
		if err != nil {
			slog.Error("parse index", "err", err)
			continue
		}
		index2Embedding[idx] = embedding
	}

	embeddings := make([][]F, len(ss))
	for i := range ss {
		embeddings[i] = index2Embedding[i]
	}

	return embeddings, err
}
