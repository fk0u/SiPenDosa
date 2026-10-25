// Command publish membuat GitHub Release untuk SiPenDosa dan mengunggah aset.
//
// Pengganti script one-off publish_v110..publish_v121 — versi dan catatan
// rilis dilewatkan lewat flag, jadi tidak perlu menyalin file per rilis.
//
// Usage:
//
//	go run ./scripts/publish -version 1.3.0 -notes release-notes.md \
//	    -assets dist/SiPenDosa-1.3.0.dmg,dist/SiPenDosa-1.3.0-Installer.pkg
//
// Token GitHub diambil otomatis dari `git credential fill`.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type releaseResponse struct {
	ID        int64  `json:"id"`
	UploadURL string `json:"upload_url"`
	HTMLURL   string `json:"html_url"`
}

func getGitToken() (string, error) {
	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader("protocol=https\nhost=github.com\n\n")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	for _, l := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(l, "password=") {
			return strings.TrimPrefix(l, "password="), nil
		}
	}
	return "", fmt.Errorf("token tidak ditemukan di git credentials")
}

func main() {
	version := flag.String("version", "", "Versi rilis tanpa prefiks v, mis. 1.3.0")
	notesFile := flag.String("notes", "", "Berkas markdown berisi catatan rilis (release body)")
	assets := flag.String("assets", "", "Daftar berkas aset yang diunggah, dipisah koma")
	flag.Parse()

	if *version == "" {
		fmt.Println("Usage: publish -version <x.y.z> [-notes file.md] [-assets a,b,c]")
		os.Exit(1)
	}
	tag := "v" + *version

	var body string
	if *notesFile != "" {
		raw, err := os.ReadFile(*notesFile)
		if err != nil {
			fmt.Printf("Gagal membaca catatan rilis: %v\n", err)
			os.Exit(1)
		}
		body = string(raw)
	} else {
		body = fmt.Sprintf("## SiPenDosa %s\n\nLihat [CHANGELOG.md](https://github.com/fk0u/SiPenDosa/blob/master/CHANGELOG.md).", tag)
	}

	token, err := getGitToken()
	if err != nil {
		fmt.Printf("Gagal mengambil token GitHub: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	auth := func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
	}

	var release releaseResponse
	checkReq, _ := http.NewRequest("GET", "https://api.github.com/repos/fk0u/SiPenDosa/releases/tags/"+tag, nil)
	auth(checkReq)
	if checkResp, err := client.Do(checkReq); err == nil {
		raw, _ := io.ReadAll(checkResp.Body)
		checkResp.Body.Close()
		if checkResp.StatusCode == http.StatusOK && json.Unmarshal(raw, &release) == nil {
			// Release sudah ada — perbarui body-nya
			payload, _ := json.Marshal(map[string]interface{}{"body": body})
			updReq, _ := http.NewRequest("PATCH", fmt.Sprintf("https://api.github.com/repos/fk0u/SiPenDosa/releases/%d", release.ID), bytes.NewBuffer(payload))
			auth(updReq)
			updReq.Header.Set("Content-Type", "application/json")
			if updResp, err := client.Do(updReq); err != nil {
				fmt.Printf("Gagal memperbarui release: %v\n", err)
				os.Exit(1)
			} else {
				updResp.Body.Close()
			}
			fmt.Printf("✓ Release %s sudah ada (ID: %d), catatan diperbarui.\n", tag, release.ID)
		}
	}

	if release.ID == 0 {
		payload, _ := json.Marshal(map[string]interface{}{
			"tag_name":         tag,
			"target_commitish": "master",
			"name":             "SiPenDosa " + tag,
			"body":             body,
			"draft":            false,
			"prerelease":       false,
		})
		req, err := http.NewRequest("POST", "https://api.github.com/repos/fk0u/SiPenDosa/releases", bytes.NewBuffer(payload))
		if err != nil {
			fmt.Printf("Error membuat request release: %v\n", err)
			os.Exit(1)
		}
		auth(req)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Gagal memanggil GitHub API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			fmt.Printf("GitHub API error (%d): %s\n", resp.StatusCode, string(raw))
			os.Exit(1)
		}
		if err := json.Unmarshal(raw, &release); err != nil {
			fmt.Printf("Gagal decode response: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ GitHub Release %s berhasil dibuat!\n  URL: %s\n", tag, release.HTMLURL)
	}

	if *assets == "" {
		fmt.Println("Tidak ada aset yang diunggah (-assets kosong). Selesai.")
		return
	}

	uploadBase := strings.Split(release.UploadURL, "{")[0]
	uploadClient := &http.Client{Timeout: 10 * time.Minute}

	for _, assetPath := range strings.Split(*assets, ",") {
		assetPath = strings.TrimSpace(assetPath)
		if assetPath == "" {
			continue
		}
		file, err := os.Open(assetPath)
		if err != nil {
			fmt.Printf("! Gagal membuka berkas %s: %v\n", assetPath, err)
			continue
		}
		stat, _ := file.Stat()
		fileName := filepath.Base(assetPath)
		fmt.Printf("==> Mengunggah %s (%.2f MB)... ", fileName, float64(stat.Size())/1024/1024)

		upReq, err := http.NewRequest("POST", fmt.Sprintf("%s?name=%s", uploadBase, fileName), file)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			file.Close()
			continue
		}
		auth(upReq)
		upReq.Header.Set("Content-Type", "application/octet-stream")
		upReq.ContentLength = stat.Size()

		upResp, err := uploadClient.Do(upReq)
		file.Close()
		if err != nil {
			fmt.Printf("Gagal kirim: %v\n", err)
			continue
		}
		if upResp.StatusCode == http.StatusCreated || upResp.StatusCode == http.StatusOK {
			fmt.Println("✓ SELESAI!")
		} else {
			respBody, _ := io.ReadAll(upResp.Body)
			fmt.Printf("Gagal (%d): %s\n", upResp.StatusCode, string(respBody))
		}
		upResp.Body.Close()
	}

	fmt.Printf("\n🎉 Selesai. Release %s: https://github.com/fk0u/SiPenDosa/releases/tag/%s\n", tag, tag)
}
