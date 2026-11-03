package federation

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRemoteFetchRejectsPrivateDestinations(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("FEDERATION_LOCAL_PEERS", "")
	client := NewHTTPClient(time.Second)
	for _, raw := range []string{"http://example.com", "https://127.0.0.1", "https://[::1]", "https://169.254.169.254/latest/meta-data", "https://10.0.0.1", "file:///etc/passwd"} {
		if r, err := client.Get(raw); err == nil {
			r.Body.Close()
			t.Errorf("private/unsafe destination accepted: %s", raw)
		}
	}
}
func TestExplicitLocalTestPeerWorksWithoutGeneralPrivateAccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	defer ts.Close()
	t.Setenv("ENV", "development")
	t.Setenv("FEDERATION_LOCAL_PEERS", ts.URL)
	r, err := NewHTTPClient(time.Second).Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r, err := NewHTTPClient(time.Second).Get("http://127.0.0.1:1"); err == nil {
		r.Body.Close()
		t.Fatal("local exception was not restricted to exact peer")
	}
}
