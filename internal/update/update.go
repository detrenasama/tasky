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
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/detrenasama/tasky/indicators/gnome"
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

// Step описывает этап процесса обновления.
type Step int

const (
	StepDownload Step = iota
	StepChecksum
	StepExtract
	StepInstall
	StepDone
)

// Reporter получает уведомления о ходе обновления. frac — доля загрузки
// (0..1) для StepDownload, либо -1, если размер неизвестен или этап не
// предполагает прогресса.
type Reporter func(step Step, msg string, frac float64)

// report вызывает rep, если он задан (nil-safe).
func report(rep Reporter, step Step, msg string, frac float64) {
	if rep != nil {
		rep(step, msg, frac)
	}
}

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

// CheckUpdate возвращает последний тег и признак необходимости обновления
// (needed=true, если опубликованная версия новее текущей).
func CheckUpdate(current string) (latest string, needed bool, err error) {
	rel, err := latestRelease()
	if err != nil {
		return "", false, err
	}
	return rel.TagName, Compare(rel.TagName, current) > 0, nil
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
// актуальна. rep получает уведомления о ходе обновления (может быть nil).
func Upgrade(current string, rep Reporter) (newVersion string, replaced bool, err error) {
	exe, err := os.Executable()
	if err != nil {
		return "", false, err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", false, err
	}
	return upgradeTo(current, exe, rep)
}

// upgradeTo — как Upgrade, но заменяет бинарник по заданному пути (exePath
// переопределяется в тестах).
func upgradeTo(current, exePath string, rep Reporter) (newVersion string, replaced bool, err error) {
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
	report(rep, StepDownload, "Загрузка "+binURL, 0)
	if err := download(binURL, tarPath, rep); err != nil {
		return "", false, err
	}
	report(rep, StepDownload, "Загружено.", 1)

	report(rep, StepChecksum, "Проверка контрольной суммы…", -1)
	want, err := fetchChecksum(tmp, sumsURL, assetName())
	if err != nil {
		return "", false, err
	}
	if got := fileSHA256(tarPath); got != want {
		return "", false, fmt.Errorf("контрольная сумма не совпадает (ожидалось %s)", want)
	}
	report(rep, StepChecksum, "Контрольная сумма совпадает.", -1)

	report(rep, StepExtract, "Распаковка…", -1)
	newBin := filepath.Join(filepath.Dir(exePath), ".tasky.new")
	if err := extractTar(tarPath, newBin); err != nil {
		return "", false, err
	}
	if err := os.Chmod(newBin, 0o755); err != nil {
		os.Remove(newBin)
		return "", false, err
	}

	report(rep, StepInstall, "Установка…", -1)
	if err := os.Rename(newBin, exePath); err != nil {
		os.Remove(newBin)
		return "", false, err
	}

	// Обновление индикатора (рядом с бинарником / в каталоге расширений).
	// Не фатально для обновления самого tasky.
	if err := updateIndicator(tarPath, exePath); err != nil {
		report(rep, StepInstall, "Индикатор: "+err.Error(), -1)
	}

	report(rep, StepDone, "Готово.", -1)
	return latest, true, nil
}

// updateIndicator обновляет системный индикатор после замены бинарника.
// Windows: всегда извлекает tasky-indicator.exe из того же релиза рядом с
// бинарником. Linux: если обнаружен GNOME и расширение уже установлено —
// перезаписывает его файлы (и пытается включить) из встроенной копии.
// Успешное обновление не шлёт отдельных уведомлений (чтобы не дублировать
// шаг установки бинарника); ошибки возвращаются вызывающему.
func updateIndicator(tarPath, exePath string) error {
	dir := filepath.Dir(exePath)
	switch runtime.GOOS {
	case "windows":
		indNew := filepath.Join(dir, ".tasky-indicator.new")
		if err := extractFile(tarPath, "tasky-indicator.exe", indNew); err != nil {
			return err
		}
		if err := os.Chmod(indNew, 0o755); err != nil {
			os.Remove(indNew)
			return err
		}
		dst := filepath.Join(dir, "tasky-indicator.exe")
		if err := os.Rename(indNew, dst); err != nil {
			os.Remove(indNew)
			return err
		}
		return nil
	case "linux":
		if !gnomeInstalled() {
			return nil
		}
		extDir := gnomeExtDir()
		if st, err := os.Stat(extDir); err != nil || !st.IsDir() {
			return nil // расширение не установлено — не трогаем
		}
		for _, f := range []string{"extension.js", "metadata.json"} {
			data, err := gnome.Files.ReadFile(f)
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(extDir, f), data, 0o644); err != nil {
				return err
			}
		}
		// Включение может потребовать перезаход в сессию — это не ошибка.
		_ = exec.Command("gnome-extensions", "enable", "tasky-indicator@detrenasama").Run()
		return nil
	}
	return nil
}

// gnomeInstalled сообщает, что ОС — Linux с оболочкой GNOME.
func gnomeInstalled() bool {
	if _, err := exec.LookPath("gnome-shell"); err != nil {
		return false
	}
	de := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	return strings.Contains(de, "gnome")
}

// gnomeExtDir возвращает каталог установки GNOME Shell-расширения.
func gnomeExtDir() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "gnome-shell", "extensions", "tasky-indicator@detrenasama")
}

// progressReader обёртывает io.Reader и сообщает о доле прочитанных данных.
type progressReader struct {
	r     io.Reader
	n     int64
	total int64
	rep   Reporter
	last  int
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.n += int64(n)
	if p.total > 0 {
		pct := int(p.n * 100 / p.total)
		if pct != p.last {
			p.last = pct
			report(p.rep, StepDownload, "", float64(p.n)/float64(p.total))
		}
	} else {
		report(p.rep, StepDownload, "", -1)
	}
	return n, err
}

func download(url, dst string, rep Reporter) error {
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
	reader := &progressReader{r: resp.Body, total: resp.ContentLength, rep: rep}
	if _, err := io.Copy(out, reader); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// fetchChecksum скачивает SHA256SUMS и возвращает хеш для файла name.
func fetchChecksum(dir, url, name string) (string, error) {
	sumsPath := filepath.Join(dir, "SHA256SUMS")
	if err := download(url, sumsPath, nil); err != nil {
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

// extractFile извлекает из tar.gz единственный файл с именем name и пишет
// его в dst. Если файла нет — ошибка.
func extractFile(tarPath, name, dst string) error {
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
		if hdr.Typeflag != tar.TypeReg || hdr.Name != name {
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
	return fmt.Errorf("в архиве нет файла %s", name)
}

// extractTar распаковывает tar.gz и записывает файл «tasky» (на Windows —
// «tasky.exe») в dst.
func extractTar(tarPath, dst string) error {
	want := "tasky"
	if runtime.GOOS == "windows" {
		want = "tasky.exe"
	}
	return extractFile(tarPath, want, dst)
}
