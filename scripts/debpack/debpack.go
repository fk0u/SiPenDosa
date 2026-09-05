package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// writeArMember writes a standard UNIX ar header and file data
func writeArMember(w *os.File, name string, data []byte) error {
	modTime := time.Now().Unix()
	hdr := fmt.Sprintf("%-16s%-12d%-6d%-6d%-8o%-10d`\n",
		name, modTime, 0, 0, 0100644, len(data))
	if len(hdr) != 60 {
		return fmt.Errorf("invalid header length for %s: %d", name, len(hdr))
	}
	if _, err := w.WriteString(hdr); err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if len(data)%2 != 0 {
		if _, err := w.WriteString("\n"); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if len(os.Args) < 5 {
		fmt.Fprintf(os.Stderr, "Usage: %s <output.deb> <debian-binary> <control.tar.gz> <data.tar.gz>\n", os.Args[0])
		os.Exit(1)
	}

	outPath := os.Args[1]
	debBinPath := os.Args[2]
	controlPath := os.Args[3]
	dataPath := os.Args[4]

	debBin, err := os.ReadFile(debBinPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading debian-binary: %v\n", err)
		os.Exit(1)
	}

	control, err := os.ReadFile(controlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading control.tar.gz: %v\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(dataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading data.tar.gz: %v\n", err)
		os.Exit(1)
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", outPath, err)
		os.Exit(1)
	}
	defer outFile.Close()

	// Ar signature
	if _, err := outFile.WriteString("!<arch>\n"); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing ar header: %v\n", err)
		os.Exit(1)
	}

	if err := writeArMember(outFile, filepath.Base(debBinPath), debBin); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", debBinPath, err)
		os.Exit(1)
	}

	if err := writeArMember(outFile, filepath.Base(controlPath), control); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", controlPath, err)
		os.Exit(1)
	}

	if err := writeArMember(outFile, filepath.Base(dataPath), data); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", dataPath, err)
		os.Exit(1)
	}

	fmt.Printf("✓ Paket deb berhasil dibuat: %s (%d bytes)\n", outPath, len(debBin)+len(control)+len(data)+188)
}
