package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type ProgressCallback func(downloaded, total int64, speed string)

func Download(ctx context.Context, url, destPath string, onProgress ProgressCallback) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength

	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tempPath := destPath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	var downloaded int64
	startTime := time.Now()
	lastUpdate := time.Now()
	buf := make([]byte, 32*1024)

	for {
		select {
		case <-ctx.Done():
			file.Close()
			os.Remove(tempPath)
			return ctx.Err()
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := file.Write(buf[:n])
			if writeErr != nil {
				file.Close()
				os.Remove(tempPath)
				return fmt.Errorf("failed to write file: %w", writeErr)
			}
			downloaded += int64(n)

			if time.Since(lastUpdate) >= 100*time.Millisecond && onProgress != nil {
				speed := calculateSpeed(downloaded, startTime)
				onProgress(downloaded, totalSize, speed)
				lastUpdate = time.Now()
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			file.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to read response: %w", err)
		}
	}

	file.Close()

	if onProgress != nil {
		speed := calculateSpeed(downloaded, startTime)
		onProgress(downloaded, totalSize, speed)
	}

	if err := os.Rename(tempPath, destPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

func calculateSpeed(downloaded int64, startTime time.Time) string {
	elapsed := time.Since(startTime).Seconds()
	if elapsed == 0 {
		return "0 B/s"
	}

	speed := float64(downloaded) / elapsed

	switch {
	case speed >= 1024*1024*1024:
		return fmt.Sprintf("%.2f GB/s", speed/(1024*1024*1024))
	case speed >= 1024*1024:
		return fmt.Sprintf("%.2f MB/s", speed/(1024*1024))
	case speed >= 1024:
		return fmt.Sprintf("%.2f KB/s", speed/1024)
	default:
		return fmt.Sprintf("%.0f B/s", speed)
	}
}
