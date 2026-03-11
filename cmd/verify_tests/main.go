package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "73477bb3-025d-43cb-8b38-49007b584531",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("secret-jwt-ket-that-we-dont-need-really"))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("media_url", "http://example.com/image.jpg")
	writer.Close()

	req, _ := http.NewRequest("POST", "http://localhost:8000/api/v1/stories", body)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Do err:", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\nBody: %s\n", resp.StatusCode, string(respBody))
}
