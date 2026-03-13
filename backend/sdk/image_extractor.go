package sdk

import (
	"backend/utils"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
)

type ImageExtractor interface {
	ExtractImage(ctx context.Context, imageReader io.Reader, arg ExtractImageArg) ([]string, []string, error)
}

type ExtractImageArg struct {
	Preprocess bool `json:"preprocess"` // 是否预处理图像
}

var _ ImageExtractor = (*AnythingImageExtractor)(nil)

type AnythingImageExtractor struct {
	// Add fields here
	URL string
}

func NewImageExtractor() *AnythingImageExtractor {
	return &AnythingImageExtractor{}
}

type ExtractResult struct {
	RecTexts  []string `json:"rec_texts"`
	RecImages []string `json:"rec_images"`
}

func (e *AnythingImageExtractor) ExtractImage(ctx context.Context, imageReader io.Reader, arg ExtractImageArg) ([]string, []string, error) {
	body, contentType, err := func() (*bytes.Buffer, string, error) {
		// 创建 buffer 用于构建 multipart 表单
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		defer writer.Close()

		// 添加文件字段
		part, err := writer.CreateFormFile("file", "file.jpg")
		if err != nil {
			return nil, "", err
		}

		// 将文件内容写入 part
		_, err = io.Copy(part, imageReader)
		if err != nil {
			return nil, "", err
		}

		part, err = writer.CreateFormField("preprocess")
		if err != nil {
			return nil, "", err
		}
		_, err = io.WriteString(part, strconv.FormatBool(arg.Preprocess))
		if err != nil {
			return nil, "", err
		}
		return body, writer.FormDataContentType(), nil
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.URL, body)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	extractRespData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	extractResp := utils.HTTPResponse[ExtractResult]{}
	err = json.Unmarshal(extractRespData, &extractResp)
	if err != nil {
		return nil, nil, err
	}

	return extractResp.Data.RecTexts, extractResp.Data.RecImages, nil
}
