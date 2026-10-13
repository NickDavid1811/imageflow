// Package s3storage stores generated artifacts in S3-compatible object storage.
package s3storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// Config contains connection details shared by MinIO, R2, and AWS S3.
type Config struct{ Endpoint, Region, Bucket, AccessKey, SecretKey string }

// Storage is an AWS Signature V4 S3-compatible artifact store.
type Storage struct {
	endpoint                             *url.URL
	region, bucket, accessKey, secretKey string
	client                               *http.Client
	now                                  func() time.Time
}

// New validates configuration and creates an S3-compatible adapter.
func New(config Config) (*Storage, error) {
	endpoint, err := url.Parse(config.Endpoint)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return nil, fmt.Errorf("invalid S3 endpoint")
	}
	if config.Region == "" || config.Bucket == "" || config.AccessKey == "" || config.SecretKey == "" {
		return nil, fmt.Errorf("incomplete S3 configuration")
	}
	return &Storage{endpoint: endpoint, region: config.Region, bucket: config.Bucket, accessKey: config.AccessKey, secretKey: config.SecretKey, client: &http.Client{Timeout: 30 * time.Second}, now: time.Now}, nil
}

// EnsureBucket creates the configured bucket when it does not exist.
func (storage *Storage) EnsureBucket(ctx context.Context) error {
	response, err := storage.do(ctx, http.MethodHead, storage.bucket, nil, "")
	if err != nil {
		return err
	}
	response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	if response.StatusCode != http.StatusNotFound {
		return responseError("check bucket", response)
	}
	response, err = storage.do(ctx, http.MethodPut, storage.bucket, nil, "")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		response.Body.Close()
		check, checkErr := storage.do(ctx, http.MethodHead, storage.bucket, nil, "")
		if checkErr == nil {
			defer check.Body.Close()
			if check.StatusCode >= 200 && check.StatusCode < 300 {
				return nil
			}
		}
		return fmt.Errorf("create bucket: S3 returned HTTP %d", response.StatusCode)
	}
	return nil
}

// Save uploads an artifact as one S3 object.
func (storage *Storage) Save(ctx context.Context, key string, data []byte) error {
	if err := validateKey(key); err != nil {
		return err
	}
	contentType := "application/octet-stream"
	if strings.HasSuffix(key, ".jpeg") {
		contentType = "image/jpeg"
	}
	if strings.HasSuffix(key, ".png") {
		contentType = "image/png"
	}
	response, err := storage.do(ctx, http.MethodPut, path.Join(storage.bucket, key), data, contentType)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return responseError("upload object", response)
	}
	return nil
}

// Open returns an object stream. Request errors may surface on the first read.
func (storage *Storage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	response, err := storage.do(ctx, http.MethodGet, path.Join(storage.bucket, key), nil, "")
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		defer response.Body.Close()
		return nil, responseError("download object", response)
	}
	return response.Body, nil
}

func (storage *Storage) do(ctx context.Context, method, objectPath string, body []byte, contentType string) (*http.Response, error) {
	requestURL := *storage.endpoint
	requestURL.Path = path.Join(storage.endpoint.Path, objectPath)
	payloadHash := hashHex(body)
	request, err := http.NewRequestWithContext(ctx, method, requestURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	now := storage.now().UTC()
	amzDate, date := now.Format("20060102T150405Z"), now.Format("20060102")
	request.Header.Set("x-amz-content-sha256", payloadHash)
	request.Header.Set("x-amz-date", amzDate)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	canonicalHeaders := "host:" + request.URL.Host + "\n" + "x-amz-content-sha256:" + payloadHash + "\n" + "x-amz-date:" + amzDate + "\n"
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalRequest := strings.Join([]string{method, request.URL.EscapedPath(), "", canonicalHeaders, signedHeaders, payloadHash}, "\n")
	scope := date + "/" + storage.region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + hashHex([]byte(canonicalRequest))
	signature := hex.EncodeToString(hmacSHA256(signingKey(storage.secretKey, date, storage.region), stringToSign))
	request.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+storage.accessKey+"/"+scope+", SignedHeaders="+signedHeaders+", Signature="+signature)
	response, err := storage.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("S3 request: %w", err)
	}
	return response, nil
}

func signingKey(secret, date, region string) []byte {
	dateKey := hmacSHA256([]byte("AWS4"+secret), date)
	regionKey := hmacSHA256(dateKey, region)
	serviceKey := hmacSHA256(regionKey, "s3")
	return hmacSHA256(serviceKey, "aws4_request")
}
func hmacSHA256(key []byte, value string) []byte {
	hash := hmac.New(sha256.New, key)
	_, _ = hash.Write([]byte(value))
	return hash.Sum(nil)
}
func hashHex(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
func responseError(action string, response *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	return fmt.Errorf("%s: S3 returned HTTP %d: %s", action, response.StatusCode, strings.TrimSpace(string(data)))
}

func validateKey(key string) error {
	if key == "" || path.Base(key) != key || strings.ContainsAny(key, `/\\`) {
		return fmt.Errorf("invalid object key")
	}
	return nil
}
