package captcha

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/big"
)

// New 生成一个 4 位数字验证码，返回 captchaId、明文 code 与 PNG 图片。
// 验证码的存储由 biz 层通过 AuthCache 端口落到 Redis。
func New() (id string, code string, pngBytes []byte, err error) {
	code, err = randomCode()
	if err != nil {
		return "", "", nil, err
	}
	buf := bytes.NewBuffer(nil)
	if err := png.Encode(buf, drawCode(code)); err != nil {
		return "", "", nil, err
	}
	return randomID(), code, buf.Bytes(), nil
}

func randomCode() (string, error) {
	b := make([]byte, 4)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b[i] = '0' + byte(n.Int64())
	}
	return string(b), nil
}

func randomID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", b)
}

// drawCode 用标准库把数字画成 PNG：白底 + 干扰线 + 噪点。
func drawCode(code string) image.Image {
	const w, h = 120, 44
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	// 干扰线
	for i := 0; i < 4; i++ {
		x1, y1 := randN(w), randN(h)
		x2, y2 := randN(w), randN(h)
		col := randColor(180)
		drawLine(img, x1, y1, x2, y2, col)
	}
	// 噪点
	for i := 0; i < 80; i++ {
		img.Set(randN(w), randN(h), randColor(200))
	}
	// 数字（用内嵌 5x7 点阵）
	digitPatterns := [10][7]string{
		{"111", "101", "101", "101", "101", "101", "111"}, // 0
		{"010", "110", "010", "010", "010", "010", "111"}, // 1
		{"111", "001", "001", "111", "100", "100", "111"}, // 2
		{"111", "001", "001", "111", "001", "001", "111"}, // 3
		{"101", "101", "101", "111", "001", "001", "001"}, // 4
		{"111", "100", "100", "111", "001", "001", "111"}, // 5
		{"111", "100", "100", "111", "101", "101", "111"}, // 6
		{"111", "001", "001", "010", "010", "010", "010"}, // 7
		{"111", "101", "101", "111", "101", "101", "111"}, // 8
		{"111", "101", "101", "111", "001", "001", "111"}, // 9
	}
	ink := color.RGBA{40, 70, 120, 255}
	x0, y0 := 12, 8
	for i, ch := range code {
		pat := digitPatterns[ch-'0']
		for r := 0; r < 7; r++ {
			for c := 0; c < 3; c++ {
				if pat[r][c] == '1' {
					px, py := x0+i*24+c*4, y0+r*4
					for dx := 0; dx < 4; dx++ {
						for dy := 0; dy < 4; dy++ {
							img.Set(px+dx+randN(1), py+dy+randN(1), ink)
						}
					}
				}
			}
		}
	}
	return img
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	dx, dy := x2-x1, y2-y1
	steps := abs(dx)
	if abs(dy) > steps {
		steps = abs(dy)
	}
	if steps == 0 {
		return
	}
	for i := 0; i <= steps; i++ {
		x := x1 + dx*i/steps
		y := y1 + dy*i/steps
		if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
			img.Set(x, y, c)
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func randN(n int) int {
	b, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(b.Int64())
}

func randColor(max uint8) color.RGBA {
	return color.RGBA{uint8(randN(int(max))), uint8(randN(int(max))), uint8(randN(int(max))), 255}
}
