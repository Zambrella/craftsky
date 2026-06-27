package api

import (
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const devMediaDir = "devmedia"

// DevMediaHandler serves screenshot/demo seed media in local development only.
// It first checks appview/devmedia for user-provided JPEG/PNG/WebP files and
// falls back to generated textile-like JPEGs so the demo seed works out of the box.
func DevMediaHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSpace(r.PathValue("name"))
		if !validDevMediaName(name) {
			http.NotFound(w, r)
			return
		}
		if served := serveDevMediaFile(w, r, name); served {
			return
		}
		serveGeneratedDevMedia(w, name)
	})
}

func validDevMediaName(name string) bool {
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func serveDevMediaFile(w http.ResponseWriter, r *http.Request, name string) bool {
	path := filepath.Join(devMediaDir, name)
	info, err := os.Stat(path)
	if err == nil && !info.IsDir() {
		http.ServeFile(w, r, path)
		return true
	}
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp"} {
		candidate := filepath.Join(devMediaDir, name+ext)
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			http.ServeFile(w, r, candidate)
			return true
		}
	}
	return false
}

func serveGeneratedDevMedia(w http.ResponseWriter, name string) {
	img := generatedDevMediaImage(name, 1200, 900)
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_ = jpeg.Encode(w, img, &jpeg.Options{Quality: 88})
}

func generatedDevMediaImage(name string, width, height int) image.Image {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	seed := h.Sum32()
	base := color.RGBA{R: uint8(80 + seed%90), G: uint8(70 + (seed>>8)%110), B: uint8(90 + (seed>>16)%100), A: 255}
	accent := color.RGBA{R: uint8(170 + (seed>>4)%70), G: uint8(140 + (seed>>12)%80), B: uint8(120 + (seed>>20)%90), A: 255}
	muted := color.RGBA{R: uint8((uint16(base.R) + 245) / 2), G: uint8((uint16(base.G) + 238) / 2), B: uint8((uint16(base.B) + 220) / 2), A: 255}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: muted}, image.Point{}, draw.Src)
	stripe := 48 + int(seed%48)
	for y := -height; y < height*2; y += stripe {
		for x := 0; x < width; x++ {
			for dy := 0; dy < stripe/3; dy++ {
				yy := y + x/3 + dy
				if yy >= 0 && yy < height {
					img.Set(x, yy, base)
				}
			}
		}
	}
	block := 120 + int((seed>>6)%80)
	for y := 0; y < height; y += block {
		for x := 0; x < width; x += block {
			if (x/block+y/block)%2 == 0 {
				draw.Draw(img, image.Rect(x, y, min(x+block/2, width), min(y+block/2, height)), &image.Uniform{C: accent}, image.Point{}, draw.Over)
			}
		}
	}
	return img
}
