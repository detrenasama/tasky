// Package update: самообновление Tasky через GitHub Releases.
package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Repo — GitHub-репозиторий релизов.
const Repo = "detrenasama/tasky"

// apiBase переопределяется в тестах на локальный httptest-сервер.
var apiBase = "https://api.github.com"

// apiBaseURL возвращает base GitHub API; TASKY_API_BASE — для сквозных тестов
// реального бинарника (локальный сервер вместо api.github.com).
func apiBaseURL() string {
	if v := os.Getenv("TASKY_API_BASE"); v != "" {
		return v
	}
	return apiBase
}

var client = &http.Client{Timeout: 15 * time.Second}

// release — минимальная модель ответа GitHub API.
type release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func latestRelease() (*release, error) {
	url := apiBaseURL() + "/repos/" + Repo + "/releases/latest"
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d от %s", resp.StatusCode, url)
	}
	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("релиз не содержит тега")
	}
	return &rel, nil
}

// TrimV убирает ведущую «v» из версии.
func TrimV(s string) string { return strings.TrimPrefix(strings.TrimSpace(s), "v") }

// Compare сравнивает версии вида X.Y.Z (ведущая «v» не обязательна):
// -1 если a < b, 0 если равны, 1 если a > b.
func Compare(a, b string) int {
	pa := parse(TrimV(a))
	pb := parse(TrimV(b))
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

func parse(s string) [3]int {
	var p [3]int
	parts := strings.Split(s, ".")
	for i := 0; i < len(parts) && i < 3; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			n = 0
		}
		p[i] = n
	}
	return p
}

// LatestVersion возвращает последний опубликованный тег (с ведущей «v»).
func LatestVersion() (string, error) {
	rel, err := latestRelease()
	if err != nil {
		return "", err
	}
	return rel.TagName, nil
}

// assetName — имя ассета с бинарником для текущей платформы.
func assetName() string {
	return fmt.Sprintf("tasky-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
}

func assetURL(rel *release, name string) (string, error) {
	for _, a := range rel.Assets {
		if a.Name == name {
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("ассет %s не найден в релизе", name)
}

// Upgrade скачивает последний релиз, проверяет SHA256 и атомарно заменяет
// текущий бинарник. Возвращает новую версию; replaced=false, если версия уже
// актуальна.
func Upgrade(current string) (newVersion string, replaced bool, err error) {
	exe, err := os.Executable()
	if err != nil {
		return "", false, err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", false, err
	}
	return upgradeTo(current, exe)
}

// upgradeTo — как Upgrade, но заменяет бинарник по заданному пути (exePath
// переопределяется в тестах).
func upgradeTo(current, exePath string) (newVersion string, replaced bool, err error) {
	if TrimV(current) == "" {
		return "", false, fmt.Errorf("неизвестная версия сборки")
	}
	rel, err := latestRelease()
	if err != nil {
		return "", false, err
	}
	latest := rel.TagName
	if Compare(latest, current) <= 0 {
		return latest, false, nil
	}

	binURL, err := assetURL(rel, assetName())
	if err != nil {
		return "", false, err
	}
	sumsURL, err := assetURL(rel, "SHA256SUMS")
	if err != nil {
		return "", false, err
	}

	tmp, err := os.MkdirTemp("", "tasky-upgrade-*")
	if err != nil {
		return "", false, err
	}
	defer os.RemoveAll(tmp)

	tarPath := filepath.Join(tmp, assetName())
	if err := download(binURL, tarPath); err != nil {
		return "", false, err
	}
	want, err := fetchChecksum(tmp, sumsURL, assetName())
	if err != nil {
		return "", false, err
	}
	if got := fileSHA256(tarPath); got != want {
		return "", false, fmt.Errorf("контрольная сумма не совпадает (ожидалось %s)", want)
	}

	newBin := filepath.Join(filepath.Dir(exePath), ".tasky.new")
	if err := extractTar(tarPath, newBin); err != nil {
		return "", false, err
	}
	if err := os.Chmod(newBin, 0o755); err != nil {
		os.Remove(newBin)
		return "", false, err
	}
	if err := os.Rename(newBin, exePath); err != nil {
		os.Remove(newBin)
		return "", false, err
	}
	return latest, true, nil
}

func download(url, dst string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d при скачивании %s", resp.StatusCode, url)
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// fetchChecksum скачивает SHA256SUMS и возвращает хеш для файла name.
func fetchChecksum(dir, url, name string) (string, error) {
	sumsPath := filepath.Join(dir, "SHA256SUMS")
	if err := download(url, sumsPath); err != nil {
		return "", err
	}
	data, err := os.ReadFile(sumsPath)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == name {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("в SHA256SUMS нет записи для %s", name)
}

func fileSHA256(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

// extractTar распаковывает tar.gz и записывает файл «tasky» в dst.
func extractTar(tarPath, dst string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg || hdr.Name != "tasky" {
			continue
		}
		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	}
	return fmt.Errorf("в архиве нет файла tasky")
}
