package pcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type PCloud struct {
	baseUrl string
	token   string
	client  *http.Client
}

func NewPCloud(url, token string, client *http.Client) PCloud {
	if url == "" {
		url = "https://api.pcloud.com"
	}
	if client == nil {
		client = http.DefaultClient
	}
	return PCloud{
		baseUrl: url,
		token:   token,
		client:  client,
	}
}

func (pc *PCloud) UserInfo() (map[string]any, error) {
	endpoint := "/userinfo"
	ctx := context.Background()
	return pc.execute(ctx, http.MethodGet, endpoint, nil, nil)
}

func (pc *PCloud) UploadFile(path, filename string, data []byte) (map[string]any, error) {
	endpoint := "/uploadfile"
	query := map[string]string{
		"path":     path,
		"filename": filename,
	}
	ctx := context.Background()
	return pc.execute(ctx, http.MethodPut, endpoint, bytes.NewBuffer(data), query)
}

func (pc *PCloud) DeleteFile(fullPath string) error {
	endpoint := "/deletefile"
	query := map[string]string{
		"path": fullPath,
	}
	ctx := context.Background()
	_, err := pc.execute(ctx, http.MethodPut, endpoint, nil, query)
	return err
}

func (pc *PCloud) OpenFile(fullPath string) (string, error) {
	endpoint := "/file_open"
	query := map[string]string{
		"flags": fmt.Sprintf("%d", 0x440),
		"path":  fullPath,
	}
	ctx := context.Background()
	data, err := pc.execute(ctx, http.MethodGet, endpoint, nil, query)
	if err != nil {
		return "", err
	}

	fd, ok := data["fd"]
	if !ok {
		return "", fmt.Errorf("unable to get file descriptor for \"%s\"", fullPath)
	}
	return fmt.Sprintf("%.0f", fd), nil
}

func (pc *PCloud) WriteFile(ctx context.Context, fd string, r io.Reader, chunkSize int) error {
	endpoint := "/file_write"
	query := map[string]string{
		"fd": fd,
	}
	if chunkSize == 0 {
		chunkSize = 1024 * 1024
	}

	size := 0
	chunk := []byte{}
	for {
		buf := make([]byte, chunkSize)
		n, err := r.Read(buf)
		if err == io.EOF {
			return nil
		}
		chunk = append(chunk, buf[:n]...)
		if len(chunk) < chunkSize {
			continue
		}
		_, err = pc.execute(ctx, http.MethodPut, endpoint, bytes.NewBuffer(chunk), query)
		if err != nil {
			return err
		}
		chunk = []byte{}
		size += n
		fmt.Printf("%s: %.2fMB\r", fd, float64(size)/(1024*1024))
	}
}

func (pc *PCloud) CloseFile(fd string) error {
	endpoint := "/file_close"
	query := map[string]string{
		"fd": fd,
	}
	ctx := context.Background()
	data, err := pc.execute(ctx, http.MethodGet, endpoint, nil, query)
	if err != nil {
		return err
	}
	fmt.Println(data)
	return nil
}

func (pc *PCloud) execute(
	ctx context.Context,
	method,
	endpoint string,
	body io.Reader,
	params map[string]string,
) (map[string]any, error) {
	url := pc.baseUrl + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	query := req.URL.Query()
	for key, value := range params {
		query.Add(key, value)
	}
	req.URL.RawQuery = query.Encode()

	req.Header.Set("Connection", "keep-alive")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", pc.token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"failed request to %s with status %s",
			url,
			resp.Status,
		)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{}
	err = json.Unmarshal(respBody, &data)
	if err != nil {
		return nil, err
	}

	result, ok := data["result"]
	if ok && result.(float64) != float64(0) {
		return nil, fmt.Errorf(
			"error on request to %s with result \"%.0f\" and error \"%s\"",
			url,
			result.(float64),
			data["error"],
		)
	}
	return data, nil
}
