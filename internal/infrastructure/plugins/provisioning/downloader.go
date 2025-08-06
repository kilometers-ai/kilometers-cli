package provisioning

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// SecurePluginDownloader handles secure plugin downloads
type SecurePluginDownloader struct {
	httpClient *http.Client
	publicKey  *rsa.PublicKey
}

// NewSecurePluginDownloader creates a new secure plugin downloader
func NewSecurePluginDownloader(publicKeyPEM string) (*SecurePluginDownloader, error) {
	// Parse public key
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return &SecurePluginDownloader{
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Longer timeout for large downloads
		},
		publicKey: rsaPub,
	}, nil
}

// DownloadPlugin downloads a plugin from a secure URL
func (d *SecurePluginDownloader) DownloadPlugin(ctx context.Context, plugin domain.ProvisionedPlugin) (io.ReadCloser, error) {
	// Create download request
	req, err := http.NewRequestWithContext(ctx, "GET", plugin.DownloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	// Add headers
	req.Header.Set("User-Agent", "kilometers-cli/1.0")

	// Send request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// VerifySignature verifies the downloaded plugin's signature
func (d *SecurePluginDownloader) VerifySignature(pluginData []byte, signature string) error {
	// Decode base64 signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Calculate hash of plugin data
	hashed := sha256.Sum256(pluginData)

	// Verify signature
	err = rsa.VerifyPKCS1v15(d.publicKey, crypto.SHA256, hashed[:], sigBytes)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// DefaultPublicKey is the Kilometers plugin signing public key
// In production, this would be embedded at build time
const DefaultPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwykbNTfMQ4vJeFwK1YwV
F3cjrDx3XTOT7aJu5F8HNvyZdPiC6kkS5K4oYrJRdHY8w9jmTfULOV7fQZFfGCq1
Zw8ZDVQxIvGxWKQvqNjqNr7WLEJ3uJV+L0fNfNxfLZNyHxHxN8nmjQJUzTwMwXZV
czv8kGpxqDdOL3nNY0S3sAie/L/3VvhHQwYFKm1W9zLjJfZsC4WJafNlvPNDXb3k
3SS9lYaKMihcHGp+3SfUwDJr5zBYqfiPD7h1F2D2xvmthVsPhe6OYp2Cx0+Dwico
dq4AZk6MN+8CZQv2DjcGOxMZ+F8pANqd5b5ErfwJEvBXXKWVhbSpGGjdJ5JkrwC0
mwIDAQAB
-----END PUBLIC KEY-----`
