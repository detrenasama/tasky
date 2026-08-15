package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v0.1.0", "v0.2.0", -1},
		{"v1.2.3", "v1.2.3", 0},
		{"0.10.0", "0.9.9", 1},
		{"v2.0.0", "v1.9.9", 1},
		{"1.2", "1.2.0", 0},
		{"dev", "dev", 0},
		{"v0.1.10", "v0.1.2", 1},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func makeTar(t *testing.T, binData []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: "tasky", Mode: 0o755, Size: int64(len(binData))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(binData); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// fakeServer поднимает httptest-сервер, эмулирующий GitHub Releases: latest,
// бинарник и SHA256SUMS.
func fakeServer(t *testing.T, tag string, binData []byte, badSum bool) *httptest.Server {
	t.Helper()
	tarBytes := makeTar(t, binData)
	good := sum(tarBytes)
	want := good
	if badSum {
		want = strings.Repeat("0", 64)
	}
	mux := http.NewServeMux()
	base := "http://"
	mux.HandleFunc("/repos/"+Repo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		u := base + r.Host
		fmt.Fprintf(w, `{"tag_name":%q,"assets":[`+
			`{"name":%q,"browser_download_url":%q},`+
			`{"name":"SHA256SUMS","browser_download_url":%q}]}`,
			tag, assetName(), u+"/bin", u+"/sums")
	})
	mux.HandleFunc("/bin", func(w http.ResponseWriter, r *http.Request) {
		w.Write(tarBytes)
	})
	mux.HandleFunc("/sums", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  %s\n", want, assetName())
	})
	srv := httptest.NewServer(mux)
	apiBase = srv.URL
	t.Cleanup(func() {
		srv.Close()
		apiBase = "https://api.github.com"
	})
	return srv
}

func TestUpgradeTo(t *testing.T) {
	binData := []byte("#!/bin/sh\necho fake tasky\n")

	t.Run("новая версия", func(t *testing.T) {
		fakeServer(t, "v2.0.0", binData, false)
		exe := filepath.Join(t.TempDir(), "tasky")
		if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
			t.Fatal(err)
		}
		next, replaced, err := upgradeTo("v1.0.0", exe)
		if err != nil {
			t.Fatal(err)
		}
		if !replaced || next != "v2.0.0" {
			t.Fatalf("replaced=%v next=%q", replaced, next)
		}
		got, err := os.ReadFile(exe)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, binData) {
			t.Errorf("бинарник не заменён: %q", got)
		}
		st, _ := os.Stat(exe)
		if st.Mode()&0o111 == 0 {
			t.Error("бинарник не исполняемый")
		}
	})

	t.Run("версия актуальна", func(t *testing.T) {
		fakeServer(t, "v1.0.0", binData, false)
		exe := filepath.Join(t.TempDir(), "tasky")
		os.WriteFile(exe, []byte("old"), 0o755)
		next, replaced, err := upgradeTo("v1.0.0", exe)
		if err != nil {
			t.Fatal(err)
		}
		if replaced || next != "v1.0.0" {
			t.Errorf("replaced=%v next=%q", replaced, next)
		}
		got, _ := os.ReadFile(exe)
		if string(got) != "old" {
			t.Error("бинарник изменён при актуальной версии")
		}
	})

	t.Run("неверная контрольная сумма", func(t *testing.T) {
		fakeServer(t, "v2.0.0", binData, true)
		exe := filepath.Join(t.TempDir(), "tasky")
		os.WriteFile(exe, []byte("old"), 0o755)
		_, _, err := upgradeTo("v1.0.0", exe)
		if err == nil {
			t.Fatal("ожидалась ошибка контрольной суммы")
		}
		got, _ := os.ReadFile(exe)
		if string(got) != "old" {
			t.Error("бинарник изменён при ошибке проверки")
		}
	})

	t.Run("нет ассета", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{"tag_name":"v2.0.0","assets":[]}`)
		}))
		apiBase = srv.URL
		defer func() {
			srv.Close()
			apiBase = "https://api.github.com"
		}()
		exe := filepath.Join(t.TempDir(), "tasky")
		os.WriteFile(exe, []byte("old"), 0o755)
		if _, _, err := upgradeTo("v1.0.0", exe); err == nil {
			t.Fatal("ожидалась ошибка отсутствия ассета")
		}
	})

	t.Run("пустая версия", func(t *testing.T) {
		if _, _, err := upgradeTo("", filepath.Join(t.TempDir(), "tasky")); err == nil {
			t.Fatal("ожидалась ошибка пустой версии")
		}
	})
}

func TestLatestVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v3.2.1","assets":[]}`)
	}))
	apiBase = srv.URL
	defer func() {
		srv.Close()
		apiBase = "https://api.github.com"
	}()
	v, err := LatestVersion()
	if err != nil {
		t.Fatal(err)
	}
	if v != "v3.2.1" {
		t.Errorf("LatestVersion() = %q", v)
	}
}
