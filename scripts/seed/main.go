package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

func main() {
	base := env("GATEWAY_URL", "http://localhost:8080")
	client := &http.Client{Timeout: 15 * time.Second}
	email := fmt.Sprintf("seed-%s@example.com", uuid.NewString()[:8])
	password := "password1"

	mustPost(client, base+"/v1/auth/register", "", map[string]string{"email": email, "password": password})
	login := mustPost(client, base+"/v1/auth/login", "", map[string]string{"email": email, "password": password})
	token := login["token"].(string)

	w1 := mustPost(client, base+"/v1/wallets", token, map[string]string{"currency": "RUB"})
	w2 := mustPost(client, base+"/v1/wallets", token, map[string]string{"currency": "RUB"})

	fmt.Printf("seeded user=%s token=%s wallet_a=%s wallet_b=%s\n", email, token, w1["wallet_id"], w2["wallet_id"])
	fmt.Println("Note: fund wallets via wallet Credit gRPC before transfers.")
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func mustPost(client *http.Client, url, token string, body any) map[string]any {
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		panic(fmt.Sprintf("%s -> %d %s", url, resp.StatusCode, raw))
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}
