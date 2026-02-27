package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/naveenspark/grimora/internal/browser"
	"github.com/naveenspark/grimora/internal/tui"
	"github.com/naveenspark/grimora/pkg/client"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// tokenFilePath returns ~/.grimora/token.
func tokenFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".grimora", "token"), nil
}

// readToken returns the auth token using precedence: env var > file > empty.
func readToken() string {
	if tok := os.Getenv("GRIMORA_TOKEN"); tok != "" {
		return tok
	}
	path, err := tokenFilePath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func run() error {
	apiURL := os.Getenv("GRIMORA_API_URL")
	if apiURL == "" {
		apiURL = "https://api.grimora.ai"
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "version", "-v":
			fmt.Println("grimora " + version)
			return nil
		case "help", "--help", "-h":
			printHelp()
			return nil
		case "terms":
			return openLegal("terms")
		case "privacy":
			return openLegal("privacy")
		case "faq":
			return openLegal("faq")
		case "login":
			return runLogin(apiURL)
		case "logout":
			return runLogout()
		case "update":
			return runUpdate()
		case "--update-done":
			if len(os.Args) >= 4 {
				printUpdateSuccess(os.Args[2], os.Args[3])
			}
			return nil
		}
	}

	token := readToken()
	if token == "" {
		printGrimoireGreeting()
		return nil
	}
	c := client.New(apiURL, token)
	// Only force re-login on actual auth failures (401), not transient errors.
	if _, err := c.GetMe(context.Background()); err != nil {
		if client.IsStatus(err, 401) {
			printGrimoireGreeting()
			return nil
		}
		// Network/server error — launch TUI anyway, it retries internally.
	}

	app := tui.NewApp(c, version)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui error: %w", err)
	}
	return nil
}

func runLogin(apiURL string) error {
	// Start ephemeral localhost server on random port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("start callback listener: %w", err)
	}
	defer listener.Close() //nolint:errcheck

	port := listener.Addr().(*net.TCPAddr).Port
	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Generate CSRF state token.
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return fmt.Errorf("generate oauth state: %w", err)
	}
	expectedState := hex.EncodeToString(stateBytes)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Verify CSRF state.
		if r.URL.Query().Get("state") != expectedState {
			http.Error(w, "invalid state", http.StatusForbidden)
			errCh <- fmt.Errorf("callback state mismatch (possible CSRF)")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			errCh <- fmt.Errorf("callback received without code")
			return
		}
		// Exchange the one-time code for the session token.
		exchangeBody, err := json.Marshal(map[string]string{"code": code})
		if err != nil {
			http.Error(w, "marshal failed", http.StatusInternalServerError)
			errCh <- fmt.Errorf("cli code exchange marshal: %w", err)
			return
		}
		exchangeResp, exchangeErr := http.Post(apiURL+"/auth/cli-exchange", "application/json",
			bytes.NewReader(exchangeBody))
		if exchangeErr != nil {
			http.Error(w, "exchange failed", http.StatusInternalServerError)
			errCh <- fmt.Errorf("cli code exchange: %w", exchangeErr)
			return
		}
		defer exchangeResp.Body.Close() //nolint:errcheck
		if exchangeResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(exchangeResp.Body, 1<<20)) //nolint:errcheck // best-effort read for error message
			http.Error(w, "exchange failed", http.StatusInternalServerError)
			errCh <- fmt.Errorf("cli code exchange: HTTP %d: %s", exchangeResp.StatusCode, string(body))
			return
		}
		var result struct {
			Token string `json:"token"`
		}
		if decErr := json.NewDecoder(exchangeResp.Body).Decode(&result); decErr != nil || result.Token == "" {
			http.Error(w, "exchange failed", http.StatusInternalServerError)
			errCh <- fmt.Errorf("cli code exchange: invalid response")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, callbackHTML) //nolint:errcheck
		tokenCh <- result.Token
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if srvErr := srv.Serve(listener); srvErr != nil && srvErr != http.ErrServerClosed {
			errCh <- srvErr
		}
	}()

	// Build login URL: use base URL (grimora.ai), not API URL (api.grimora.ai).
	baseURL := os.Getenv("GRIMORA_BASE_URL")
	if baseURL == "" {
		u, _ := url.Parse(apiURL) //nolint:errcheck // apiURL is already validated or defaulted
		host := u.Hostname()
		if strings.HasPrefix(host, "api.") {
			u.Host = strings.TrimPrefix(host, "api.")
			if u.Port() != "" {
				u.Host += ":" + u.Port()
			}
		}
		baseURL = u.String()
	}
	loginParams := url.Values{}
	loginParams.Set("cli_port", strconv.Itoa(port))
	loginParams.Set("state", expectedState)
	loginURL := baseURL + "/auth/github/login?" + loginParams.Encode()

	fmt.Printf("Opening browser to authenticate...\n")
	if err := browser.Open(loginURL); err != nil {
		fmt.Printf("Could not open browser. Visit this URL manually:\n  %s\n", loginURL)
	}

	// Wait for callback or timeout.
	select {
	case tok := <-tokenCh:
		// Shut down the server.
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(shutCtx) //nolint:errcheck

		// Save token.
		tokPath, err := tokenFilePath()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(tokPath), 0700); err != nil {
			return fmt.Errorf("create ~/.grimora dir: %w", err)
		}
		if err := os.WriteFile(tokPath, []byte(tok), 0600); err != nil {
			return fmt.Errorf("save token: %w", err)
		}

		// Verify by calling /api/me.
		c := client.New(apiURL, tok)
		me, err := c.GetMe(context.Background())
		if err != nil {
			fmt.Printf("Token saved but verification failed: %v\n", err)
			return nil
		}
		fmt.Printf("Authenticated as @%s\n\n", me.GitHubLogin)

		// Launch TUI automatically after login.
		app := tui.NewApp(c, version)
		p := tea.NewProgram(app, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("tui error: %w", err)
		}
		return nil

	case srvErr := <-errCh:
		return fmt.Errorf("callback server error: %w", srvErr)

	case <-time.After(2 * time.Minute):
		return fmt.Errorf("login timed out — no callback received within 2 minutes")
	}
}

func runLogout() error {
	tokPath, err := tokenFilePath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(tokPath); os.IsNotExist(err) {
		fmt.Println("Already logged out.")
		return nil
	}
	if err := os.Remove(tokPath); err != nil {
		return fmt.Errorf("remove token: %w", err)
	}
	fmt.Println("Logged out.")
	return nil
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// isNewerVersion returns true if latest is a newer semver than current.
func isNewerVersion(latest, current string) bool {
	parse := func(v string) (int, int, int) {
		v = strings.TrimPrefix(v, "v")
		parts := strings.SplitN(v, ".", 3)
		atoi := func(s string) int {
			n, _ := strconv.Atoi(s) //nolint:errcheck // zero-value on parse failure is desired
			return n
		}
		var maj, min, patch int
		if len(parts) > 0 {
			maj = atoi(parts[0])
		}
		if len(parts) > 1 {
			min = atoi(parts[1])
		}
		if len(parts) > 2 {
			patch = atoi(parts[2])
		}
		return maj, min, patch
	}
	lMaj, lMin, lPatch := parse(latest)
	cMaj, cMin, cPatch := parse(current)
	if lMaj != cMaj {
		return lMaj > cMaj
	}
	if lMin != cMin {
		return lMin > cMin
	}
	return lPatch > cPatch
}

func runUpdate() error {
	if version == "dev" {
		fmt.Println("dev build — install a release to enable updates")
		return nil
	}

	// Resolve the real binary path (follow symlinks).
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("runUpdate: find executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("runUpdate: resolve symlinks: %w", err)
	}

	// Fetch latest release from GitHub.
	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Get("https://api.github.com/repos/naveenspark/grimora/releases/latest")
	if err != nil {
		return fmt.Errorf("runUpdate: check for updates: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("runUpdate: GitHub API returned %s", resp.Status)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("runUpdate: parse release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(version, "v")
	if !isNewerVersion(latestVersion, currentVersion) {
		printAlreadyCurrent("v" + currentVersion)
		return nil
	}

	// Find the right asset for this platform.
	tarballName := fmt.Sprintf("grimora_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	var tarballURL, checksumsURL string
	for _, a := range release.Assets {
		switch a.Name {
		case tarballName:
			tarballURL = a.BrowserDownloadURL
		case "checksums.txt":
			checksumsURL = a.BrowserDownloadURL
		}
	}
	if tarballURL == "" {
		return fmt.Errorf("runUpdate: no asset %s in release %s", tarballName, release.TagName)
	}

	// Download to temp dir.
	tmpDir, err := os.MkdirTemp("", "grimora-update-*")
	if err != nil {
		return fmt.Errorf("runUpdate: create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	tarballPath := filepath.Join(tmpDir, tarballName)
	if err := downloadFile(httpClient, tarballURL, tarballPath); err != nil {
		return fmt.Errorf("runUpdate: download tarball: %w", err)
	}

	// Verify checksum (mandatory).
	if checksumsURL == "" {
		return fmt.Errorf("runUpdate: release missing checksums.txt — aborting update")
	}
	checksumsPath := filepath.Join(tmpDir, "checksums.txt")
	if err := downloadFile(httpClient, checksumsURL, checksumsPath); err != nil {
		return fmt.Errorf("runUpdate: download checksums: %w", err)
	}
	if err := verifyChecksum(tarballPath, checksumsPath, tarballName); err != nil {
		return fmt.Errorf("runUpdate: %w", err)
	}

	// Extract the grimora binary from the tarball.
	newBinaryPath := filepath.Join(tmpDir, "grimora")
	if err := extractBinary(tarballPath, newBinaryPath); err != nil {
		return fmt.Errorf("runUpdate: extract: %w", err)
	}

	// Atomic replace: write to .new, then rename over the original.
	stagePath := execPath + ".new"
	defer os.Remove(stagePath) //nolint:errcheck

	src, err := os.Open(newBinaryPath)
	if err != nil {
		return fmt.Errorf("runUpdate: open extracted binary: %w", err)
	}
	defer src.Close() //nolint:errcheck

	dst, err := os.OpenFile(stagePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied writing to %s — try with sudo", filepath.Dir(execPath))
		}
		return fmt.Errorf("runUpdate: create staged binary: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close() //nolint:errcheck
		return fmt.Errorf("runUpdate: write staged binary: %w", err)
	}
	if err := dst.Close(); err != nil {
		return fmt.Errorf("runUpdate: close staged binary: %w", err)
	}

	if err := os.Rename(stagePath, execPath); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied replacing %s — try with sudo", execPath)
		}
		return fmt.Errorf("runUpdate: replace binary: %w", err)
	}

	// Re-exec into the NEW binary so its updated code renders the success message.
	// The running process still has the old code in memory after os.Rename.
	execErr := syscall.Exec(execPath, []string{"grimora", "--update-done", "v" + currentVersion, "v" + latestVersion}, os.Environ())
	if execErr != nil {
		// Fallback if exec fails (e.g., Windows).
		printUpdateSuccess("v"+currentVersion, "v"+latestVersion)
	}
	return nil
}

func downloadFile(client *http.Client, url, dest string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %s from %s", resp.Status, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()                   //nolint:errcheck
	const maxDownloadSize = 100 << 20 // 100 MB
	_, err = io.Copy(f, io.LimitReader(resp.Body, maxDownloadSize))
	return err
}

func verifyChecksum(filePath, checksumsPath, fileName string) error {
	data, err := os.ReadFile(checksumsPath)
	if err != nil {
		return fmt.Errorf("read checksums: %w", err)
	}
	var expected string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, fileName) {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				expected = parts[0]
				break
			}
		}
	}
	if expected == "" {
		return fmt.Errorf("no checksum found for %s", fileName)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file for checksum: %w", err)
	}
	defer f.Close() //nolint:errcheck

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash file: %w", err)
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

func extractBinary(tarballPath, dest string) error {
	f, err := os.Open(tarballPath)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close() //nolint:errcheck

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}
		// Only extract the grimora binary; ignore everything else.
		if filepath.Base(hdr.Name) == "grimora" && hdr.Typeflag == tar.TypeReg {
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			const maxBinarySize = 200 << 20 // 200 MB
			if _, err := io.Copy(out, io.LimitReader(tr, maxBinarySize)); err != nil {
				out.Close() //nolint:errcheck
				return err
			}
			return out.Close()
		}
	}
	return fmt.Errorf("grimora binary not found in tarball")
}

func openLegal(page string) error {
	url := "https://grimora.ai/" + page
	if err := browser.Open(url); err != nil {
		fmt.Println(url)
	}
	return nil
}

const callbackHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Grimora</title>
<style>
@import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600;700&display=swap');
*{margin:0;padding:0;box-sizing:border-box}
body{
  background:#0a0a10;color:#e4e4ec;
  font-family:'JetBrains Mono','SF Mono','Consolas',monospace;
  height:100vh;display:flex;align-items:center;justify-content:center;
  overflow:hidden;
}
.card{text-align:center;position:relative}
.logo{
  font-size:32px;font-weight:700;letter-spacing:12px;
  text-transform:uppercase;margin-bottom:24px;
}
.logo span{display:inline-block;animation:shimmer 3s ease-in-out infinite}
.logo span:nth-child(1){color:#7c3aed;animation-delay:0s}
.logo span:nth-child(2){color:#8b5cf6;animation-delay:.1s}
.logo span:nth-child(3){color:#a78bfa;animation-delay:.2s}
.logo span:nth-child(4){color:#c4b5fd;animation-delay:.3s}
.logo span:nth-child(5){color:#a78bfa;animation-delay:.4s}
.logo span:nth-child(6){color:#8b5cf6;animation-delay:.5s}
.logo span:nth-child(7){color:#7c3aed;animation-delay:.6s}
@keyframes shimmer{
  0%,100%{opacity:.6;transform:translateY(0)}
  50%{opacity:1;transform:translateY(-2px)}
}
.check{
  width:48px;height:48px;margin:0 auto 20px;
  border:2px solid #34d474;border-radius:50%;
  display:flex;align-items:center;justify-content:center;
  animation:pop .4s cubic-bezier(.175,.885,.32,1.275) forwards;
  opacity:0;
}
@keyframes pop{
  0%{opacity:0;transform:scale(0)}
  100%{opacity:1;transform:scale(1)}
}
.check svg{width:24px;height:24px}
.msg{
  font-size:14px;color:#34d474;font-weight:600;
  margin-bottom:8px;
  animation:fadein .6s .3s forwards;opacity:0;
}
.sub{
  font-size:12px;color:#505868;
  animation:fadein .6s .5s forwards;opacity:0;
}
@keyframes fadein{0%{opacity:0;transform:translateY(4px)}100%{opacity:1;transform:translateY(0)}}
.particles{position:fixed;inset:0;pointer-events:none;overflow:hidden}
.p{
  position:absolute;width:2px;height:2px;border-radius:50%;
  background:#34d474;opacity:0;
  animation:drift 4s ease-out forwards;
}
@keyframes drift{
  0%{opacity:.8;transform:translateY(0) scale(1)}
  100%{opacity:0;transform:translateY(-120px) scale(0)}
}
</style>
</head>
<body>
<div class="particles" id="particles"></div>
<div class="card">
  <div class="logo">
    <span>G</span><span>R</span><span>I</span><span>M</span><span>O</span><span>R</span><span>A</span>
  </div>
  <div class="check">
    <svg viewBox="0 0 24 24" fill="none" stroke="#34d474" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
      <polyline points="20 6 9 17 4 12"/>
    </svg>
  </div>
  <div class="msg">authenticated</div>
  <div class="sub">return to your terminal</div>
</div>
<script>
(function(){
  var c=document.getElementById('particles');
  for(var i=0;i<12;i++){
    var d=document.createElement('div');
    d.className='p';
    d.style.left=Math.random()*100+'%';
    d.style.bottom=Math.random()*30+'%';
    d.style.animationDelay=Math.random()*2+'s';
    d.style.background=['#34d474','#d4a844','#7c3aed','#c084e0'][i%4];
    c.appendChild(d);
  }
})();
</script>
</body>
</html>`
