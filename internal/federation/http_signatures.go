package federation

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SignRequest signs an outgoing HTTP request using HTTP Signatures (draft-cavage)
// Signs: (request-target), host, date, digest (for POST bodies)
func SignRequest(req *http.Request, privateKey *rsa.PrivateKey, keyID string) error {
	// Set Date header if not present
	if req.Header.Get("Date") == "" {
		req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	}

	// Build signed headers list
	headers := []string{"(request-target)", "host", "date"}

	// For POST requests, add digest
	if req.Body != nil && (req.Method == "POST" || req.Method == "PUT") {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body.Close()

		// Compute SHA-256 digest
		hash := sha256.Sum256(bodyBytes)
		digest := "SHA-256=" + base64.StdEncoding.EncodeToString(hash[:])
		req.Header.Set("Digest", digest)
		headers = append(headers, "digest")

		// Reset body for actual sending
		req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		req.ContentLength = int64(len(bodyBytes))
	}

	// Build signing string
	var signingParts []string
	for _, h := range headers {
		switch h {
		case "(request-target)":
			signingParts = append(signingParts, fmt.Sprintf("(request-target): %s %s", strings.ToLower(req.Method), req.URL.Path))
		case "host":
			signingParts = append(signingParts, fmt.Sprintf("host: %s", req.Host))
		case "date":
			signingParts = append(signingParts, fmt.Sprintf("date: %s", req.Header.Get("Date")))
		case "digest":
			signingParts = append(signingParts, fmt.Sprintf("digest: %s", req.Header.Get("Digest")))
		}
	}
	signingString := strings.Join(signingParts, "\n")

	// Sign with RSA-SHA256
	hashed := sha256.Sum256([]byte(signingString))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	// Build Signature header
	sigHeader := fmt.Sprintf(
		`keyId="%s",algorithm="rsa-sha256",headers="%s",signature="%s"`,
		keyID,
		strings.Join(headers, " "),
		base64.StdEncoding.EncodeToString(signature),
	)
	req.Header.Set("Signature", sigHeader)

	return nil
}

// VerifyRequest verifies the HTTP Signature on an incoming request
func VerifyRequest(req *http.Request, publicKeyPEM string) error {
	sigHeader := req.Header.Get("Signature")
	if sigHeader == "" {
		return fmt.Errorf("no Signature header")
	}

	// Parse signature header
	params := parseSignatureHeader(sigHeader)
	sigB64, ok := params["signature"]
	if !ok {
		return fmt.Errorf("missing signature value")
	}
	headersStr, ok := params["headers"]
	if !ok {
		return fmt.Errorf("missing headers param")
	}

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Build signing string from the listed headers
	headerList := strings.Split(headersStr, " ")
	var signingParts []string
	for _, h := range headerList {
		switch h {
		case "(request-target)":
			signingParts = append(signingParts, fmt.Sprintf("(request-target): %s %s", strings.ToLower(req.Method), req.URL.Path))
		case "host":
			host := req.Host
			if host == "" {
				host = req.Header.Get("Host")
			}
			signingParts = append(signingParts, fmt.Sprintf("host: %s", host))
		case "date":
			signingParts = append(signingParts, fmt.Sprintf("date: %s", req.Header.Get("Date")))
		case "digest":
			signingParts = append(signingParts, fmt.Sprintf("digest: %s", req.Header.Get("Digest")))
		}
	}
	signingString := strings.Join(signingParts, "\n")

	// Parse public key
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return fmt.Errorf("failed to decode public key PEM")
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}
	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not RSA")
	}

	// Verify
	hashed := sha256.Sum256([]byte(signingString))
	err = rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA256, hashed[:], sigBytes)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// parseSignatureHeader parses a Signature header into key-value pairs
func parseSignatureHeader(header string) map[string]string {
	params := make(map[string]string)
	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		eqIdx := strings.Index(part, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:eqIdx])
		val := strings.TrimSpace(part[eqIdx+1:])
		val = strings.Trim(val, "\"")
		params[key] = val
	}
	return params
}
