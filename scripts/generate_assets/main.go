package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// Brand Colors (Zero Purple/Blue Policy)
var (
	colorObsidian   = color.RGBA{R: 9, G: 10, B: 15, A: 255}       // #090a0f
	colorSurface    = color.RGBA{R: 19, G: 21, B: 31, A: 255}     // #13151f
	colorScarlet    = color.RGBA{R: 225, G: 29, B: 72, A: 255}    // #e11d48
	colorRoseLight  = color.RGBA{R: 244, G: 63, B: 94, A: 255}    // #f43f5e
	colorGold       = color.RGBA{R: 245, G: 158, B: 11, A: 255}   // #f59e0b
	colorAmberLight = color.RGBA{R: 251, G: 191, B: 36, A: 255}   // #fbbf24
	colorEmerald    = color.RGBA{R: 16, G: 185, B: 129, A: 255}   // #10b981
	colorWhite      = color.RGBA{R: 248, G: 250, B: 252, A: 255}  // #f8fafc
	colorGrayMuted  = color.RGBA{R: 148, G: 163, B: 184, A: 255}  // #94a3b8
)

// blend mixes c1 and c2 by factor t (0.0 .. 1.0)
func blend(c1, c2 color.RGBA, t float64) color.RGBA {
	if t <= 0 {
		return c1
	}
	if t >= 1 {
		return c2
	}
	return color.RGBA{
		R: uint8(float64(c1.R)*(1-t) + float64(c2.R)*t),
		G: uint8(float64(c1.G)*(1-t) + float64(c2.G)*t),
		B: uint8(float64(c1.B)*(1-t) + float64(c2.B)*t),
		A: uint8(float64(c1.A)*(1-t) + float64(c2.A)*t),
	}
}

// Generate Icon of arbitrary square dimension (for 16, 32, 180, 192, 512, 1024)
func renderIconImage(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	center := float64(size) / 2.0
	scale := float64(size) / 1024.0
	cornerRad := float64(size) * 0.22

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx := float64(x)
			fy := float64(y)

			dx := math.Abs(fx - center)
			dy := math.Abs(fy - center)
			maxDist := float64(size)*0.5 - (32.0 * scale)

			rx := math.Max(0, dx-(maxDist-cornerRad))
			ry := math.Max(0, dy-(maxDist-cornerRad))
			cornerDist := math.Hypot(rx, ry)

			if cornerDist > cornerRad {
				img.Set(x, y, color.Transparent)
				continue
			}

			distFromCenter := math.Hypot(fx-center, fy-center)
			t := distFromCenter / (float64(size) * 0.65)
			bg := blend(colorSurface, colorObsidian, t)

			// Border highlight
			edgeDist := cornerRad - cornerDist
			borderThick := math.Max(1.0, 8.0*scale)
			if rx > 0 || ry > 0 {
				if edgeDist < borderThick {
					bg = blend(bg, colorGold, (borderThick-edgeDist)/borderThick*0.6)
				}
			} else {
				dEdge := math.Min(maxDist-dx, maxDist-dy)
				if dEdge < borderThick {
					bg = blend(bg, colorScarlet, (borderThick-dEdge)/borderThick*0.6)
				}
			}

			// Render Flame Core (scaled)
			flameCX := center
			flameCY := center + (30.0 * scale)

			fdx := fx - flameCX
			fdy := fy - flameCY

			flameTop := -220.0 * scale
			flameBot := 200.0 * scale

			if fdy > flameTop && fdy < flameBot {
				flameWidth := (flameBot - fdy) * 0.65
				if fdy < 0 {
					ratio := (math.Abs(flameTop) + fdy) / math.Abs(flameTop)
					if ratio > 0 {
						flameWidth = 140.0 * scale * math.Pow(ratio, 1.4)
					}
				}
				if math.Abs(fdx) < flameWidth && flameWidth > 0 {
					flameT := math.Abs(fdx) / flameWidth
					flameCol := blend(colorAmberLight, colorScarlet, flameT)
					if math.Abs(fdx) < flameWidth*0.45 && fdy > -80*scale && fdy < 140*scale {
						flameCol = blend(colorWhite, colorGold, math.Abs(fdx)/(flameWidth*0.45))
					}
					bg = blend(bg, flameCol, (1.0 - flameT*0.7))
				}
			}

			// Crown at top
			crownTop := 280.0 * scale
			crownBot := 340.0 * scale
			crownHalfW := 120.0 * scale

			if fy >= crownTop && fy <= crownBot && math.Abs(fx-center) <= crownHalfW {
				cWidth := crownHalfW - (fy-crownTop)*0.5
				if math.Abs(fx-center) <= cWidth {
					crownCol := blend(colorGold, colorAmberLight, (fy-crownTop)/(60.0*scale))
					bg = blend(bg, crownCol, 0.9)
				}
			}

			// WhatsApp Badge in bottom-right
			badgeCX := center + 248.0*scale
			badgeCY := center + 248.0*scale
			badgeRadius := 130.0 * scale

			badgeDist := math.Hypot(fx-badgeCX, fy-badgeCY)
			if badgeDist <= badgeRadius {
				ringThick := math.Max(1.0, 15.0*scale)
				if badgeDist > badgeRadius-ringThick {
					bg = colorObsidian
				} else {
					bg = blend(colorEmerald, color.RGBA{R: 5, G: 150, B: 105, A: 255}, badgeDist/badgeRadius)
					pdx := math.Abs(fx - badgeCX)
					pdy := math.Abs(fy - badgeCY)
					pLimit := 48.0 * scale
					pInner := 12.0 * scale
					if pdx+pdy < pLimit && (pdx > pInner || pdy > pInner) {
						bg = colorWhite
					}
				}
			}

			img.Set(x, y, bg)
		}
	}
	return img
}

func savePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// Write .ico file embedding a PNG payload
func saveICO(path string, img image.Image) error {
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return err
	}
	pngData := pngBuf.Bytes()

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w > 255 {
		w = 0 // 0 means 256 in ICO spec
	}
	if h > 255 {
		h = 0
	}

	var icoBuf bytes.Buffer
	// ICONDIR (6 bytes)
	binary.Write(&icoBuf, binary.LittleEndian, uint16(0)) // Reserved
	binary.Write(&icoBuf, binary.LittleEndian, uint16(1)) // Type 1 = ICO
	binary.Write(&icoBuf, binary.LittleEndian, uint16(1)) // Image count = 1

	// ICONDIRENTRY (16 bytes)
	icoBuf.WriteByte(byte(w))
	icoBuf.WriteByte(byte(h))
	icoBuf.WriteByte(0)                                              // Color count (0 = >=8bpp)
	icoBuf.WriteByte(0)                                              // Reserved
	binary.Write(&icoBuf, binary.LittleEndian, uint16(1))            // Color planes
	binary.Write(&icoBuf, binary.LittleEndian, uint16(32))           // Bits per pixel
	binary.Write(&icoBuf, binary.LittleEndian, uint32(len(pngData))) // Image bytes size
	binary.Write(&icoBuf, binary.LittleEndian, uint32(22))           // Offset (6 + 16 = 22)

	// Image data
	icoBuf.Write(pngData)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, icoBuf.Bytes(), 0644)
}

// Generate Open Graph / Twitter Preview Image (1200x630)
func generateOGPreview(outPath string) error {
	const width = 1200
	const height = 630
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Render small icon to place on the left (400x400)
	iconImg := renderIconImage(400)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fx := float64(x)
			fy := float64(y)

			// Dark gradient canvas
			tX := fx / float64(width)
			tY := fy / float64(height)
			bg := blend(colorObsidian, colorSurface, tY*0.6+tX*0.2)

			// Scarlet glow at top-left
			glow1 := math.Hypot(fx-250, fy-250)
			if glow1 < 380 {
				bg = blend(bg, colorScarlet, (380-glow1)/380.0*0.22)
			}

			// Gold glow at bottom-right
			glow2 := math.Hypot(fx-950, fy-450)
			if glow2 < 420 {
				bg = blend(bg, colorGold, (420-glow2)/420.0*0.14)
			}

			// Decorative top bar (4px)
			if fy <= 6 {
				prog := fx / float64(width)
				barCol := blend(colorScarlet, colorGold, prog)
				bg = barCol
			}

			// Horizontal divider near bottom (y = 540)
			if fy >= 540 && fy <= 542 && fx >= 80 && fx <= 1120 {
				fade := math.Sin((fx - 80) / 1040.0 * math.Pi)
				bg = blend(bg, colorGold, fade*0.4)
			}

			img.Set(x, y, bg)
		}
	}

	// Paste icon on left side (x: 100 .. 500, y: 115 .. 515)
	iconOffsetX := 100
	iconOffsetY := 115
	for iy := 0; iy < 400; iy++ {
		for ix := 0; ix < 400; ix++ {
			iconCol := iconImg.RGBAAt(ix, iy)
			if iconCol.A > 0 {
				targetX := iconOffsetX + ix
				targetY := iconOffsetY + iy
				if targetX < width && targetY < height {
					bgCol := img.RGBAAt(targetX, targetY)
					alpha := float64(iconCol.A) / 255.0
					composite := blend(bgCol, iconCol, alpha)
					img.Set(targetX, targetY, composite)
				}
			}
		}
	}

	return savePNG(outPath, img)
}

// Generate DMG Background (1280x880)
func generateDMGBackground(outPath string) error {
	const width = 1280
	const height = 880
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fx := float64(x)
			fy := float64(y)

			tY := fy / float64(height)
			bg := blend(colorObsidian, colorSurface, tY*0.7)

			glow1Dist := math.Hypot(fx-200, fy-150)
			if glow1Dist < 450 {
				intensity := (450 - glow1Dist) / 450.0 * 0.18
				bg = blend(bg, colorScarlet, intensity)
			}

			glow2Dist := math.Hypot(fx-1080, fy-150)
			if glow2Dist < 450 {
				intensity := (450 - glow2Dist) / 450.0 * 0.15
				bg = blend(bg, colorGold, intensity)
			}

			// Left Drop Target Halo (x=320, y=480, r=160) - SiPenDosa.app
			targetLeftDist := math.Hypot(fx-320, fy-480)
			if targetLeftDist > 140 && targetLeftDist < 165 {
				intensity := (1.0 - math.Abs(targetLeftDist-152.5)/12.5) * 0.4
				bg = blend(bg, colorScarlet, intensity)
			}

			// Right Drop Target Halo (x=960, y=480, r=160) - Applications
			targetRightDist := math.Hypot(fx-960, fy-480)
			if targetRightDist > 140 && targetRightDist < 165 {
				intensity := (1.0 - math.Abs(targetRightDist-152.5)/12.5) * 0.4
				bg = blend(bg, colorGold, intensity)
			}

			// Connecting curved arrow indicator
			if fx >= 460 && fx <= 820 {
				arrowProg := (fx - 460) / 360.0
				curveY := 480.0 - math.Sin(arrowProg*math.Pi)*50.0
				dy := math.Abs(fy - curveY)
				if dy < 6.0 {
					arrowColor := blend(colorScarlet, colorGold, arrowProg)
					bg = blend(bg, arrowColor, (6.0-dy)/6.0*0.7)
				}
				if fx > 780 && fx < 820 {
					headDy1 := math.Abs(fy - (curveY - (fx - 780)*0.6))
					headDy2 := math.Abs(fy - (curveY + (fx - 780)*0.6))
					if headDy1 < 5.0 || headDy2 < 5.0 {
						bg = blend(bg, colorGold, 0.8)
					}
				}
			}

			if fy >= 760 && fy <= 762 && fx >= 100 && fx <= 1180 {
				fade := math.Sin((fx - 100) / 1080.0 * math.Pi)
				lineCol := blend(colorSurface, colorGold, fade*0.4)
				bg = blend(bg, lineCol, 0.6)
			}

			img.Set(x, y, bg)
		}
	}

	return savePNG(outPath, img)
}

func main() {
	fmt.Println("==> Men-generate seluruh aset visual SiPenDosa (Retina, PWA, Favicon, OG Preview)...")

	// 1. macOS Assets
	appIcon1024 := renderIconImage(1024)
	savePNG("assets/macos/AppIcon.png", appIcon1024)
	generateDMGBackground("assets/macos/dmg_background.png")

	// 2. Web & PWA Assets
	savePNG("web/static/img/icon-512.png", renderIconImage(512))
	savePNG("web/static/img/icon-192.png", renderIconImage(192))
	savePNG("web/static/img/apple-touch-icon.png", renderIconImage(180))
	fav64 := renderIconImage(64)
	savePNG("web/static/img/favicon.png", fav64)
	saveICO("web/static/favicon.ico", fav64)

	// 3. Open Graph Preview (1200x630)
	generateOGPreview("web/static/img/og-preview.png")

	fmt.Println("✓ Seluruh aset visual (Web, PWA, Favicon, DMG) berhasil dibuat!")
}
