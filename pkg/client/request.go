package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
)

func requestApi[ReplyBody any, RequestBody any](ctx context.Context, c *Client, method string, p string, body RequestBody) (*ReplyBody, error) {
	return requestApi2[ReplyBody, RequestBody](ctx, c, method, p, body, true)
}

func requestApi2[ReplyBody any, RequestBody any](ctx context.Context, c *Client, method string, p string, body RequestBody, withToken bool) (*ReplyBody, error) {
	if withToken {
		err := c.RefreshToken(ctx)
		if err != nil {
			return nil, err
		}
	}

	u, err := url.Parse(c.url)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, p)

	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(b)))

	if withToken && c.clientAuth.Token != nil {
		req.Header.Set("Authorization", "Bearer "+c.clientAuth.Token.AccessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s request returned http status %s", p, resp.Status)
	}

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var reply ReplyBody
	err = json.Unmarshal(b, &reply)
	if err != nil {
		return nil, err
	}

	return &reply, nil
}
