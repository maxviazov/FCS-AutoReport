// SVG to appicon.png for Wails (build/appicon.png). Run from repo root: go run scripts/svg2appicon/main.go
package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

const size = 512

func main() {
	repoRoot := "."
	if len(os.Args) > 1 {
		repoRoot = os.Args[1]
	}
	svgPath := filepath.Join(repoRoot, "build", "icon.svg")
	outPath := filepath.Join(repoRoot, "build", "appicon.png")

	f, err := os.Open(svgPath)
	if err != nil {
		panic("open svg: " + err.Error())
	}
	defer f.Close()

	icon, err := oksvg.ReadIconStream(f, oksvg.IgnoreErrorMode)
	if err != nil {
		panic("parse svg: " + err.Error())
	}

	icon.SetTarget(0, 0, size, size)
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	dasher := rasterx.NewDasher(size, size, scanner)
	icon.Draw(dasher, 1.0)

	outDir := filepath.Dir(outPath)
	_ = os.MkdirAll(outDir, 0755)
	w, err := os.Create(outPath)
	if err != nil {
		panic("create png: " + err.Error())
	}
	defer w.Close()
	if err := png.Encode(w, img); err != nil {
		panic("encode png: " + err.Error())
	}
	println("Written:", outPath)
}
