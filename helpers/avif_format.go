package helpers

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Go has no native AVIF decoder. ffmpeg (already a runtime dependency for
// video transcoding) decodes AVIF via libdav1d, so it is reused here as the
// codec backend instead of adding a cgo dependency on libavif.
//
// Registering through image.RegisterFormat makes AVIF transparently decodable
// everywhere the stdlib image package is used (image.Decode/DecodeConfig and,
// through them, imaging.Open/Decode and LoadImageWithOrientation).
func init() {
	image.RegisterFormat("avif", "????ftypavif", decodeAVIF, decodeConfigAVIF)
	image.RegisterFormat("avif", "????ftypavis", decodeAVIF, decodeConfigAVIF)
}

const (
	avifMaxInputBytes  = 32 << 20 // guards the temp file against a mislabeled huge upload
	avifProcessTimeout = 20 * time.Second
)

func decodeConfigAVIF(r io.Reader) (image.Config, error) {
	srcPath, cleanup, err := bufferAVIFToTempFile(r)
	if err != nil {
		return image.Config{}, err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), avifProcessTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=p=0:s=x", srcPath).Output()
	if err != nil {
		return image.Config{}, fmt.Errorf("avif: probe failed: %w", err)
	}
	width, height, err := parseFFProbeDimensions(out)
	if err != nil {
		return image.Config{}, err
	}
	return image.Config{ColorModel: color.NRGBAModel, Width: width, Height: height}, nil
}

func decodeAVIF(r io.Reader) (image.Image, error) {
	srcPath, cleanupSrc, err := bufferAVIFToTempFile(r)
	if err != nil {
		return nil, err
	}
	defer cleanupSrc()

	dst, err := os.CreateTemp("", "avif-decode-*.png")
	if err != nil {
		return nil, err
	}
	dstPath := dst.Name()
	_ = dst.Close()
	defer os.Remove(dstPath)

	ctx, cancel := context.WithTimeout(context.Background(), avifProcessTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", srcPath, "-frames:v", "1", "-update", "1", dstPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("avif: decode failed: %w: %s", err, stderr.String())
	}

	f, err := os.Open(dstPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

// bufferAVIFToTempFile drains r (the format registry hands decoders the
// unread-ahead stream, not a path) into a temp file ffmpeg/ffprobe can open.
func bufferAVIFToTempFile(r io.Reader) (path string, cleanup func(), err error) {
	tmp, err := os.CreateTemp("", "avif-src-*.avif")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { os.Remove(tmp.Name()) }

	n, copyErr := io.Copy(tmp, io.LimitReader(r, avifMaxInputBytes+1))
	closeErr := tmp.Close()
	if copyErr != nil {
		cleanup()
		return "", nil, copyErr
	}
	if closeErr != nil {
		cleanup()
		return "", nil, closeErr
	}
	if n > avifMaxInputBytes {
		cleanup()
		return "", nil, fmt.Errorf("avif: input exceeds %d bytes", avifMaxInputBytes)
	}
	return tmp.Name(), cleanup, nil
}

func parseFFProbeDimensions(out []byte) (int, int, error) {
	s := strings.TrimSpace(string(out))
	parts := strings.SplitN(s, "x", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("avif: unexpected ffprobe output %q", s)
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return width, height, nil
}
