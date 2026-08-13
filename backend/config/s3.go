package config

import (
	"fmt"
	"net/url"
	"strings"
)

func (c *S3Config) BucketName() string {
	if c.Bucket != "" {
		return c.Bucket
	}
	return "static-file"
}

func (c *S3Config) UseSecure() bool {
	if c.UseSSL {
		return true
	}
	endpoint := strings.ToLower(c.Endpoint)
	return strings.Contains(endpoint, "aliyuncs.com") ||
		strings.Contains(endpoint, "amazonaws.com") ||
		strings.HasPrefix(strings.ToLower(c.PublicBaseURL), "https://")
}

func (c *S3Config) PublicObjectURL(objectKey string) string {
	key := strings.TrimPrefix(objectKey, "/")
	if base := strings.TrimSuffix(strings.TrimSpace(c.PublicBaseURL), "/"); base != "" {
		return base + "/" + key
	}
	scheme := "http"
	if c.UseSecure() {
		scheme = "https"
	}
	endpoint := strings.TrimPrefix(strings.TrimPrefix(c.Endpoint, "https://"), "http://")
	return fmt.Sprintf("%s://%s.%s/%s", scheme, c.BucketName(), endpoint, key)
}

// StaticFileProxyTarget returns reverse-proxy target for /static-file/* (scheme + host).
func (c *S3Config) StaticFileProxyTarget() (*url.URL, error) {
	raw := strings.TrimSpace(c.PublicBaseURL)
	if raw == "" {
		scheme := "http"
		if c.UseSecure() {
			scheme = "https"
		}
		raw = fmt.Sprintf("%s://%s.%s", scheme, c.BucketName(), c.Endpoint)
	}
	return url.Parse(raw)
}

func (c *S3Config) IsExternalObjectStorage() bool {
	endpoint := strings.ToLower(c.Endpoint)
	return !strings.Contains(endpoint, "minio") && !strings.HasSuffix(endpoint, ":9000")
}

// ResolveStaticFilePath turns app-relative paths (/static-file/...) into a fetchable object URL.
func (c *S3Config) ResolveStaticFilePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	const appPrefix = "/static-file/"
	if strings.HasPrefix(path, appPrefix) {
		return c.PublicObjectURL(strings.TrimPrefix(path, appPrefix))
	}
	bucketPath := "/" + c.BucketName() + "/"
	if strings.HasPrefix(path, bucketPath) {
		return c.PublicObjectURL(strings.TrimPrefix(path, bucketPath))
	}
	return c.PublicObjectURL(strings.TrimPrefix(path, "/"))
}
