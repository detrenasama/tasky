package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRegisterStaticServesIndexAndAssets(t *testing.T) {
	fsys := fstest.MapFS{
		"web/dist/index.html":       &fstest.MapFile{Data: []byte("<html>app</html>")},
		"web/dist/assets/app.js":    &fstest.MapFile{Data: []byte("console.log(1)")},
		"web/dist/assets/style.css": &fstest.MapFile{Data: []byte("body{}")},
	}
	mux := http.NewServeMux()
	RegisterStatic(mux, fsys)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	get := func(path string) (int, string, string) {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, resp.Header.Get("Content-Type"), string(b)
	}

	code, ct, body := get("/")
	if code != 200 || body != "<html>app</html>" {
		t.Fatalf("/: %d %q", code, body)
	}
	if ct != "text/html; charset=utf-8" {
		t.Fatalf("/ content-type = %q", ct)
	}

	_, ct, body = get("/assets/app.js")
	if !strings.Contains(ct, "javascript") {
		t.Fatalf("/assets/app.js content-type = %q", ct)
	}
	if body != "console.log(1)" {
		t.Fatalf("/assets/app.js body = %q", body)
	}

	// SPA-фоллбэк: неизвестный маршрут без расширения отдаёт index.html.
	_, _, body = get("/tasks/42")
	if body != "<html>app</html>" {
		t.Fatalf("SPA fallback body = %q", body)
	}

	// Несуществующий файл с расширением — 404.
	code, _, _ = get("/assets/missing.png")
	if code != 404 {
		t.Fatalf("/assets/missing.png code = %d, ожидался 404", code)
	}
}

func TestRegisterStaticNilFSServesPlaceholder(t *testing.T) {
	mux := http.NewServeMux()
	RegisterStatic(mux, nil)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !strings.Contains(string(b), "Веб-интерфейс не собран") {
		t.Fatalf("placeholder: %d %q", resp.StatusCode, string(b))
	}
}
