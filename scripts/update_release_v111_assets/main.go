package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ReleaseInfo struct {
	ID        int64  `json:"id"`
	UploadURL string `json:"upload_url"`
	Assets    []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"assets"`
}

func getGitToken() (string, error) {
	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader("protocol=https\nhost=github.com\n")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "password=") {
			return strings.TrimPrefix(l, "password="), nil
		}
	}
	return "", fmt.Errorf("token not found in git credentials")
}

func main() {
	token, err := getGitToken()
	if err != nil {
		fmt.Printf("Gagal mengambil token GitHub: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 300 * time.Second}

	// 1. Dapatkan Release v1.1.1
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/fk0u/SiPenDosa/releases/tags/v1.1.1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error GET release: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var rel ReleaseInfo
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &rel); err != nil {
		fmt.Printf("Error unmarshal release: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Ditemukan Release v1.1.1 ID: %d dengan %d assets\n", rel.ID, len(rel.Assets))

	// Target files to replace
	targetFiles := []string{
		"SiPenDosa-Android.apk",
		"SiPenDosa-Android.aab",
	}

	for _, target := range targetFiles {
		// Hapus asset lama jika ada
		for _, a := range rel.Assets {
			if a.Name == target {
				fmt.Printf("==> Menghapus asset lama: %s (ID: %d)...\n", a.Name, a.ID)
				delReq, _ := http.NewRequest("DELETE", fmt.Sprintf("https://api.github.com/repos/fk0u/SiPenDosa/releases/assets/%d", a.ID), nil)
				delReq.Header.Set("Authorization", "Bearer "+token)
				delResp, delErr := client.Do(delReq)
				if delErr == nil {
					delResp.Body.Close()
					fmt.Printf("    Berhasil menghapus %s\n", a.Name)
				}
			}
		}

		// Upload asset baru
		localPath := filepath.Join("dist", target)
		fileInfo, err := os.Stat(localPath)
		if err != nil {
			fmt.Printf("Berkas lokal %s tidak ditemukan: %v\n", localPath, err)
			continue
		}

		file, err := os.Open(localPath)
		if err != nil {
			fmt.Printf("Gagal membuka berkas %s: %v\n", localPath, err)
			continue
		}

		uploadURL := strings.Split(rel.UploadURL, "{")[0] + "?name=" + target
		fmt.Printf("==> Mengunggah berkas baru %s (%d bytes)...\n", target, fileInfo.Size())

		upReq, _ := http.NewRequest("POST", uploadURL, file)
		upReq.Header.Set("Authorization", "Bearer "+token)
		upReq.Header.Set("Content-Type", "application/octet-stream")
		upReq.ContentLength = fileInfo.Size()

		upResp, upErr := client.Do(upReq)
		if upErr != nil {
			file.Close()
			fmt.Printf("Error uploading %s: %v\n", target, upErr)
			continue
		}

		upBody, _ := io.ReadAll(upResp.Body)
		upResp.Body.Close()
		file.Close()

		if upResp.StatusCode == 201 {
			fmt.Printf("✓ Sukses mengunggah %s ke Release v1.1.1!\n", target)
		} else {
			fmt.Printf("Gagal upload %s (HTTP %d): %s\n", target, upResp.StatusCode, string(upBody))
		}
	}
}
