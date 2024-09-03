package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
)

var DurationRegex = regexp.MustCompile(`size=.*time=(.*):(.*):(.*)\..*bitrate=.*speed=.*`)

func ProcessAudio(ctx context.Context, reader io.Reader) (string, int64, error) {
	number, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return "", 0, fmt.Errorf("failed to generate random number: %w", err)
	}

	targetFileName := filepath.Join(os.TempDir(), fmt.Sprintf("audio-%d.mp3", number.Uint64()))

	cmd := exec.CommandContext(ctx, "./ffmpeg", "-hide_banner", "-nostats", "-vn", "-i", "-", targetFileName)
	cmd.Stdin = reader
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", 0, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", 0, fmt.Errorf("failed to start processing audio: %w", err)
	}

	stderrBytes, err := io.ReadAll(stderr)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read stderr: %w", err)
	}

	durationMatch := DurationRegex.FindSubmatch(stderrBytes)
	if len(durationMatch) != 4 {
		return "", 0, fmt.Errorf("failed to find duration: `%v`", string(stderrBytes))
	}

	hours, err := strconv.ParseInt(string(durationMatch[1]), 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse hours: %w", err)
	}

	minutes, err := strconv.ParseInt(string(durationMatch[2]), 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse minutes: %w", err)
	}

	seconds, err := strconv.ParseInt(string(durationMatch[3]), 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse seconds: %w", err)
	}

	durationSeconds := hours*3600 + minutes*60 + seconds

	if err := cmd.Wait(); err != nil {
		return "", 0, fmt.Errorf("failed to run ffmpeg: %w", err)
	}

	if durationSeconds <= 0 {
		return "", 0, fmt.Errorf("failed to find positive duration: `%v`", string(stderrBytes))
	}

	return targetFileName, durationSeconds, nil
}

func ProcessImage(ctx context.Context, reader io.Reader) (string, error) {
	number, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return "", fmt.Errorf("failed to generate random number: %w", err)
	}

	targetFileName := filepath.Join(os.TempDir(), fmt.Sprintf("audio-%d.webp", number.Uint64()))

	cmd := exec.CommandContext(ctx, "./ffmpeg", "-hide_banner", "-nostats", "-i", "-", targetFileName)
	cmd.Stdin = reader

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run ffmpeg: %w", err)
	}

	return targetFileName, nil
}
