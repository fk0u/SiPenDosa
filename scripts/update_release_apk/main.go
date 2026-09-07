package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
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

	client := &http.Client{}

	// 1. Dapatkan Release v1.0.0
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/fk0u/SiPenDosa/releases/tags/v1.0.0", nil)
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
		fmt.Printf("Error decode JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Release ID: %d\n", rel.ID)

	uploadFile := func(fileName, contentType string) {
		filePath := "dist/" + fileName
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("! File %s tidak ditemukan: %v\n", filePath, err)
			return
		}
		defer file.Close()

		stat, _ := file.Stat()
		sizeMB := float64(stat.Size()) / 1024 / 1024

		// Hapus asset lama jika ada
		for _, asset := range rel.Assets {
			if asset.Name == fileName {
				fmt.Printf("==> Menghapus asset lama: %s (ID: %d)... ", asset.Name, asset.ID)
				delReq, _ := http.NewRequest("DELETE", fmt.Sprintf("https://api.github.com/repos/fk0u/SiPenDosa/releases/assets/%d", asset.ID), nil)
				delReq.Header.Set("Authorization", "Bearer "+token)
				delResp, err := client.Do(delReq)
				if err == nil && (delResp.StatusCode == http.StatusNoContent || delResp.StatusCode == http.StatusOK) {
					fmt.Printf("✓ Terhapus\n")
				} else {
					fmt.Printf("Status: %d\n", delResp.StatusCode)
				}
				delResp.Body.Close()
			}
		}

		uploadBase := strings.Split(rel.UploadURL, "{")[0]
		targetURL := fmt.Sprintf("%s?name=%s", uploadBase, fileName)

		fmt.Printf("==> Mengunggah %s (%.2f MB) ke GitHub Release... ", fileName, sizeMB)
		upReq, err := http.NewRequest("POST", targetURL, file)
		if err != nil {
			fmt.Printf("Error buat request: %v\n", err)
			return
		}
		upReq.Header.Set("Authorization", "Bearer "+token)
		upReq.Header.Set("Content-Type", contentType)
		upReq.ContentLength = stat.Size()

		upResp, err := client.Do(upReq)
		if err != nil {
			fmt.Printf("Error upload: %v\n", err)
			return
		}
		defer upResp.Body.Close()

		if upResp.StatusCode == http.StatusCreated || upResp.StatusCode == http.StatusOK {
			fmt.Printf("✓ SELESAI!\n")
		} else {
			upBody, _ := io.ReadAll(upResp.Body)
			fmt.Printf("GAGAL (%d): %s\n", upResp.StatusCode, string(upBody))
		}
	}

	uploadFile("SiPenDosa-Android.apk", "application/vnd.android.package-archive")
	uploadFile("SiPenDosa-Android.aab", "application/octet-stream")

	fmt.Println("\n🎉 Berkas SiPenDosa Android (APK & AAB Universal API 21+) berhasil diperbarui di GitHub Release v1.0.0!")
}
