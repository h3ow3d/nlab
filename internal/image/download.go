// Package image handles downloading and verifying the Ubuntu base cloud image.
package image

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/h3ow3d/nlab/internal/log"
)

const (
	baseDir      = "/var/lib/libvirt/images"
	baseImage    = baseDir + "/ubuntu-base.qcow2"
	imageName    = "jammy-server-cloudimg-amd64.img"
	imageURL     = "https://cloud-images.ubuntu.com/jammy/current/" + imageName
	sumsURL      = "https://cloud-images.ubuntu.com/jammy/current/SHA256SUMS"
	downloadTimeout = 20 * time.Minute // generous for a ~600 MB image
)

// Download downloads, verifies, and installs the Ubuntu 22.04 base cloud image
// into /var/lib/libvirt/images/ubuntu-base.qcow2.  It mirrors download_base.sh.
func Download() error {
	if _, err := os.Stat(baseImage); err == nil {
		log.Skip("Base image already exists")
		return nil
	}

	if err := downloadImage(); err != nil {
		return err
	}
	if err := verifyChecksum(); err != nil {
		_ = os.Remove("ubuntu.img")
		return err
	}
	return installImage()
}

func downloadImage() error {
	log.Info("Downloading Ubuntu cloud image")
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	if err := downloadFile(ctx, imageURL, "ubuntu.img"); err != nil {
		return fmt.Errorf("download image: %w", err)
	}
	if err := downloadFile(ctx, sumsURL, "ubuntu.SHA256SUMS"); err != nil {
		return fmt.Errorf("download SHA256SUMS: %w", err)
	}
	return nil
}

func verifyChecksum() error {
	log.Info("Verifying checksum")

	expected, err := readExpectedChecksum("ubuntu.SHA256SUMS", imageName)
	_ = os.Remove("ubuntu.SHA256SUMS")
	if err != nil {
		return err
	}

	actual, err := sha256File("ubuntu.img")
	if err != nil {
		return fmt.Errorf("checksum ubuntu.img: %w", err)
	}

	if actual != expected {
		return fmt.Errorf("checksum mismatch – expected %s but got %s", expected, actual)
	}

	log.Ok("Checksum OK")
	return nil
}

func installImage() error {
	log.Info("Moving image to libvirt storage")

	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return fmt.Errorf("create image dir: %w", err)
	}

	// Use os.Rename; fall back to copy+remove for cross-device moves.
	if err := os.Rename("ubuntu.img", baseImage); err != nil {
		if err2 := copyFile("ubuntu.img", baseImage); err2 != nil {
			return fmt.Errorf("install image: %w", err2)
		}
		_ = os.Remove("ubuntu.img")
	}

	log.Ok("Base image ready at " + baseImage)
	log.Info("Run: sudo chown libvirt-qemu:kvm " + baseImage)
	return nil
}

// downloadFile saves a URL to dest using the provided context.
func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// readExpectedChecksum finds the SHA-256 for filename in a SHA256SUMS file.
func readExpectedChecksum(sumsFile, filename string) (string, error) {
	f, err := os.Open(sumsFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, filename) {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				return parts[0], nil
			}
		}
	}
	return "", fmt.Errorf("checksum not found for %s in %s", filename, sumsFile)
}

// sha256File computes the hex SHA-256 of a file.
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// copyFile copies src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
