package client

import (
	"context"
	"fmt"

	"github.com/dboxed/dboxed-volume/pkg/server/models"
)

func (c *Client) S3ProxyListObjects(ctx context.Context, repoId int64, req models.S3ProxyListObjectsRequest) (*models.S3ProxyListObjectsResult, error) {
	return requestApi[models.S3ProxyListObjectsResult](ctx, c, "POST", fmt.Sprintf("v1/repositories/%d/s3proxy/list-objects", repoId), req)
}

func (c *Client) S3ProxyPresignPut(ctx context.Context, repoId int64, req models.S3ProxyPresignPutRequest) (*models.S3ProxyPresignPutResult, error) {
	return requestApi[models.S3ProxyPresignPutResult](ctx, c, "POST", fmt.Sprintf("v1/repositories/%d/s3proxy/presign-put", repoId), req)
}

func (c *Client) S3ProxyRenameObject(ctx context.Context, repoId int64, req models.S3ProxyRenameObjectRequest) (*models.S3ProxyRenameObjectResult, error) {
	return requestApi[models.S3ProxyRenameObjectResult](ctx, c, "POST", fmt.Sprintf("v1/repositories/%d/s3proxy/rename-object", repoId), req)
}

func (c *Client) S3ProxyDeleteObject(ctx context.Context, repoId int64, req models.S3ProxyDeleteObjectRequest) (*models.S3ProxyDeleteObjectResult, error) {
	return requestApi[models.S3ProxyDeleteObjectResult](ctx, c, "POST", fmt.Sprintf("v1/repositories/%d/s3proxy/delete-object", repoId), req)
}
