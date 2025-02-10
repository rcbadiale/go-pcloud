package go_pcloud

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

func (pc *PCloud) ListFolder(path string) ([]string, error) {
	endpoint := "/listfolder"
	query := map[string]string{
		"path": path,
	}
	ctx := context.Background()
	response, err := pc.execute(ctx, http.MethodGet, endpoint, nil, query)
	if err != nil {
		return nil, err
	}
	// convert the response to a list of paths
	content, ok := response["metadata"].(map[string]any)["contents"]
	if !ok {
		return nil, fmt.Errorf("unable to get content for \"%s\"", path)
	}
	output := []string{}
	for _, item := range content.([]interface{}) {
		item := item.(map[string]any)
		output = append(output, item["path"].(string))
	}
	return output, nil
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

func (pc *PCloud) FileStat(fullPath string) (map[string]any, error) {
	endpoint := "/stat"
	query := map[string]string{
		"path": fullPath,
	}
	ctx := context.Background()
	return pc.execute(ctx, http.MethodGet, endpoint, nil, query)
}

func (pc *PCloud) ReadFile(ctx context.Context, fd string, chunkSize int) ([]byte, error) {
	endpoint := "/file_read"
	query := map[string]string{
		"fd":    fd,
		"count": fmt.Sprintf("%d", chunkSize),
	}
	return pc.executeRaw(ctx, http.MethodGet, endpoint, nil, query)
}

func (pc *PCloud) WriteFile(ctx context.Context, fd string, buf []byte) (int, error) {
	endpoint := "/file_write"
	query := map[string]string{
		"fd": fd,
	}
	response, err := pc.execute(ctx, http.MethodPut, endpoint, bytes.NewReader(buf), query)
	if err != nil {
		return 0, nil
	}
	n, ok := response["bytes"].(float64)
	if !ok {
		return 0, fmt.Errorf("unable to get written bytes")
	}
	return int(n), nil
}

func (pc *PCloud) CloseFile(fd string) error {
	endpoint := "/file_close"
	query := map[string]string{
		"fd": fd,
	}
	ctx := context.Background()
	_, err := pc.execute(ctx, http.MethodGet, endpoint, nil, query)
	if err != nil {
		return err
	}
	return nil
}

func (pc *PCloud) executeRaw(
	ctx context.Context,
	method,
	endpoint string,
	body io.Reader,
	params map[string]string,
) ([]byte, error) {
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
	return io.ReadAll(resp.Body)
}

func (pc *PCloud) execute(
	ctx context.Context,
	method,
	endpoint string,
	body io.Reader,
	params map[string]string,
) (map[string]any, error) {
	respBody, err := pc.executeRaw(ctx, method, endpoint, body, params)
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
			endpoint,
			result.(float64),
			data["error"],
		)
	}
	return data, nil
}
