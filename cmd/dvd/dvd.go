// dvd.go
package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"

	"golang.org/x/term"
)

const fps = 30
const extraWidth = 4 // widen box by +4 columns

func main() {
	rand.Seed(time.Now().UnixNano())

	cleanupTerminal := startTerminalSession()
	defer cleanupTerminal()

	ctx, keyCh, release := watchForExit(context.Background())
	defer release()

	runDVD(ctx, keyCh)
}

func runDVD(ctx context.Context, key <-chan struct{}) {
	content := []string{
		"    D V D",
		"    --o--",
		"    video",
	}
	mode := newDVDMode(content, 1)
	mode.Run(ctx, key)
}

func startTerminalSession() func() {
	fmt.Print("\x1b[?1049h\x1b[?25l")
	oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))

	return func() {
		if oldState != nil {
			_ = term.Restore(int(os.Stdin.Fd()), oldState)
		}
		fmt.Print("\x1b[0m\x1b[2J\x1b[H\x1b[?25h\x1b[?1049l")
	}
}

func watchForExit(parent context.Context) (context.Context, <-chan struct{}, func()) {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt)
	keyCh := make(chan struct{}, 1)
	go func() {
		b := make([]byte, 1)
		_, _ = os.Stdin.Read(b)
		select {
		case keyCh <- struct{}{}:
		default:
		}
	}()
	return ctx, keyCh, func() { stop() }
}

type dvdMode struct {
	content    []string
	pad        int
	boxWidth   int
	boxHeight  int
	screenW    int
	screenH    int
	arenaW     int
	arenaH     int
	x, y       float64
	vx, vy     float64
	color      int
	cornerHits int
	last       time.Time
}

func newDVDMode(content []string, pad int) *dvdMode {
	cw, ch := textSize(content)
	bw := cw + 2*pad + 2 + extraWidth
	bh := ch + 2*pad + 2

	w, h := termSizeOr(120, 30)
	aw, ah := max(0, w-bw), max(0, h-bh)

	speed := 35.0
	angle := 0.82

	return &dvdMode{
		content:   content,
		pad:       pad,
		boxWidth:  bw,
		boxHeight: bh,
		screenW:   w,
		screenH:   h,
		arenaW:    aw,
		arenaH:    ah,
		x:         rand.Float64() * float64(aw),
		y:         rand.Float64() * float64(ah),
		vx:        speed * math.Cos(angle),
		vy:        speed * math.Sin(angle),
		color:     ansiColorRand(),
		last:      time.Now(),
	}
}

func (m *dvdMode) Run(ctx context.Context, key <-chan struct{}) {
	ticker := time.NewTicker(time.Second / fps)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-key:
			return
		case <-ticker.C:
			m.resizeIfNeeded()
			dt := m.stepTime(time.Now())
			if m.advance(dt) {
				m.handleCorner()
			}
			m.draw()
		}
	}
}

func (m *dvdMode) resizeIfNeeded() {
	w, h := termSizeOr(120, 30)
	if w == m.screenW && h == m.screenH {
		return
	}
	m.screenW, m.screenH = w, h
	m.arenaW = max(0, w-m.boxWidth)
	m.arenaH = max(0, h-m.boxHeight)
	if m.x > float64(m.arenaW) {
		m.x = float64(m.arenaW)
	}
	if m.y > float64(m.arenaH) {
		m.y = float64(m.arenaH)
	}
}

func (m *dvdMode) stepTime(now time.Time) float64 {
	dt := now.Sub(m.last).Seconds()
	if dt <= 0 {
		dt = 1.0 / fps
	}
	m.last = now
	return dt
}

func (m *dvdMode) advance(dt float64) bool {
	nx := m.x + m.vx*dt
	ny := m.y + m.vy*dt

	tx := math.Inf(1)
	if m.vx < 0 {
		tx = (0 - m.x) / (m.vx * dt)
	} else if m.vx > 0 {
		tx = (float64(m.arenaW) - m.x) / (m.vx * dt)
	}
	ty := math.Inf(1)
	if m.vy < 0 {
		ty = (0 - m.y) / (m.vy * dt)
	} else if m.vy > 0 {
		ty = (float64(m.arenaH) - m.y) / (m.vy * dt)
	}

	corner := false

	if tx >= 0 && tx <= 1 && ty >= 0 && ty <= 1 && math.Abs(tx-ty) <= 1e-3 {
		corner = true
		m.x += m.vx * dt * tx
		m.y += m.vy * dt * ty
		m.vx, m.vy = -m.vx, -m.vy
		rem := (1 - ((tx + ty) / 2)) * dt
		m.x += m.vx * rem
		m.y += m.vy * rem
	} else {
		if nx < 0 {
			nx = -nx
			m.vx = -m.vx
		} else if nx > float64(m.arenaW) {
			nx = 2*float64(m.arenaW) - nx
			m.vx = -m.vx
		}
		if ny < 0 {
			ny = -ny
			m.vy = -m.vy
		} else if ny > float64(m.arenaH) {
			ny = 2*float64(m.arenaH) - ny
			m.vy = -m.vy
		}
		m.x, m.y = nx, ny
	}

	ix := int(math.Round(m.x)) + 1
	iy := int(math.Round(m.y)) + 1
	if (ix == 1 || ix == m.screenW-m.boxWidth+1) && (iy == 1 || iy == m.screenH-m.boxHeight+1) {
		corner = true
	}
	if (approx(m.x, 0, 1e-3) || approx(m.x, float64(m.arenaW), 1e-3)) &&
		(approx(m.y, 0, 1e-3) || approx(m.y, float64(m.arenaH), 1e-3)) {
		corner = true
	}

	return corner
}

func (m *dvdMode) handleCorner() {
	prev := m.color
	for m.color == prev {
		m.color = ansiColorRand()
	}
	m.cornerHits++
}

func (m *dvdMode) draw() {
	fmt.Print("\x1b[2J\x1b[H")
	move(1, 1)
	fmt.Printf("Corner hits: %d", m.cornerHits)

	top := int(math.Round(m.y)) + 1
	left := int(math.Round(m.x)) + 1
	textColor, backgroundColor := colorsForBlock(m.color)
	drawSolidBlock(top, left, m.boxWidth, m.boxHeight, m.pad, m.content, textColor, backgroundColor)
	fmt.Print("\x1b[0m")
}

// ---- drawing / util ----

func drawSolidBlock(top, left, w, h, pad int, lines []string, textColor, backgroundColor int) {
	colorSequence := fmt.Sprintf("\x1b[%d;%dm", textColor, backgroundColor)
	fill := strings.Repeat(" ", w)

	for row := 0; row < h; row++ {
		move(left, top+row)
		fmt.Print(colorSequence)
		fmt.Print(fill)
	}

	row := top + 1 + pad
	for _, line := range lines {
		move(left+1+pad, row)
		fmt.Print(colorSequence)
		fmt.Print(line)
		row++
	}
}

func colorsForBlock(baseColor int) (textColor, backgroundColor int) {
	backgroundColor = backgroundFromBase(baseColor)
	textColor = readableTextColor(baseColor)
	return
}

func backgroundFromBase(baseColor int) int {
	return baseColor + 10
}

func readableTextColor(baseColor int) int {
	switch baseColor {
	case 33, 36,
		91, 92, 93, 94, 95, 96:
		return 30
	default:
		return 97
	}
}

func move(col, row int) {
	if col < 1 {
		col = 1
	}
	if row < 1 {
		row = 1
	}
	fmt.Printf("\x1b[%d;%dH", row, col)
}

func textSize(lines []string) (w, h int) {
	h = len(lines)
	for _, s := range lines {
		if ln := runeLen(s); ln > w {
			w = ln
		}
	}
	return
}

func runeLen(s string) (n int) {
	for range s {
		n++
	}
	return
}

func termSizeOr(dw, dh int) (int, int) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return dw, dh
	}
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return dw, dh
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func approx(v, target, eps float64) bool {
	return math.Abs(v-target) <= eps
}

func ansiColorRand() int {
	palette := []int{31, 32, 33, 34, 35, 36, 91, 92, 93, 94, 95, 96}
	return palette[rand.Intn(len(palette))]
}
