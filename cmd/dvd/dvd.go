// dvd.go
package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"golang.org/x/term"
)

const fps = 30
const extraWidth = 4 // widen box by +4 columns

func main() {
	rand.Seed(time.Now().UnixNano())

	// Alt screen & hide cursor
	fmt.Print("\x1b[?1049h\x1b[?25l")
	defer fmt.Print("\x1b[0m\x1b[2J\x1b[H\x1b[?25h\x1b[?1049l")

	// Any key exits
	oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	keyCh := make(chan struct{}, 1)
	go func() { b := make([]byte, 1); _, _ = os.Stdin.Read(b); keyCh <- struct{}{} }()

	// 3 lines inside UTF-8 box (4 leading spaces)
	content := []string{
		"    D V D",
		"    --o--",
		"    video",
	}
	pad := 1
	cw, ch := textSize(content)
	bw := cw + 2*pad + 2 + extraWidth // borders + extra width
	bh := ch + 2*pad + 2

	// Arena for top-left of box: [0..aw] x [0..ah]
	W, H := termSizeOr(120, 30)
	aw, ah := max(0, W-bw), max(0, H-bh)

	// Motion
	speed := 35.0
	angle := 0.82
	x := rand.Float64() * float64(aw)
	y := rand.Float64() * float64(ah)
	vx := speed * math.Cos(angle)
	vy := speed * math.Sin(angle)

	color := ansiColorRand()
	cornerHits := 0

	tick := time.NewTicker(time.Second / fps)
	defer tick.Stop()
	last := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-keyCh:
			return
		case <-tick.C:
			// Resize + recompute arena
			if w2, h2 := termSizeOr(120, 30); w2 != W || h2 != H {
				W, H = w2, h2
				aw, ah = max(0, W-bw), max(0, H-bh)
				if x > float64(aw) { x = float64(aw) }
				if y > float64(ah) { y = float64(ah) }
			}

			// Step
			now := time.Now()
			dt := now.Sub(last).Seconds()
			if dt <= 0 { dt = 1.0 / fps }
			last = now

			nx := x + vx*dt
			ny := y + vy*dt

			// Continuous collision timing (detects simultaneous corner within frame)
			const eps = 1e-6
			tx := math.Inf(1)
			if vx < 0 {
				tx = (0 - x) / (vx * dt)
			} else if vx > 0 {
				tx = (float64(aw) - x) / (vx * dt)
			}
			ty := math.Inf(1)
			if vy < 0 {
				ty = (0 - y) / (vy * dt)
			} else if vy > 0 {
				ty = (float64(ah) - y) / (vy * dt)
			}

			corner := false

			if tx >= 0 && tx <= 1 && ty >= 0 && ty <= 1 && math.Abs(tx-ty) <= 1e-3 {
				// exact simultaneous wall impact this frame
				corner = true
				x += vx * dt * tx
				y += vy * dt * ty
				vx, vy = -vx, -vy
				rem := (1 - ((tx + ty) / 2)) * dt
				x += vx * rem
				y += vy * rem
			} else {
				// Per-axis reflect to exact edges
				if nx < 0 {
					nx = -nx; vx = -vx
				} else if nx > float64(aw) {
					nx = 2*float64(aw) - nx; vx = -vx
				}
				if ny < 0 {
					ny = -ny; vy = -vy
				} else if ny > float64(ah) {
					ny = 2*float64(ah) - ny; vy = -vy
				}
				x, y = nx, ny
			}

			// Rendered-corner check (robust for any terminal size + rounding)
			ix := int(math.Round(x)) + 1
			iy := int(math.Round(y)) + 1
			if (ix == 1 || ix == W-bw+1) && (iy == 1 || iy == H-bh+1) {
				corner = true
			}
			// Also check float-space edges in case W/H changed mid-frame
			if (approx(x, 0, 1e-3) || approx(x, float64(aw), 1e-3)) &&
				(approx(y, 0, 1e-3) || approx(y, float64(ah), 1e-3)) {
				corner = true
			}

			// Corner → color change + counter
			if corner {
				prev := color
				for color == prev {
					color = ansiColorRand()
				}
				cornerHits++
			}

			// Draw
			fmt.Print("\x1b[2J\x1b[H")
			// persistent counter (background)
			move(1, 1)
			fmt.Printf("Corner hits: %d", cornerHits)

			// box
			fmt.Printf("\x1b[%dm", color)
			top := iy
			left := ix
			drawUnicodeBox(top, left, bw, bh)
			drawTextInBox(top, left, pad, content)
			fmt.Print("\x1b[0m")
		}
	}
}

// ---- drawing / util ----

func drawUnicodeBox(top, left, w, h int) {
	move(left, top); fmt.Print("┌")
	for i := 0; i < w-2; i++ { fmt.Print("─") }
	fmt.Print("┐")
	for r := 1; r < h-1; r++ {
		move(left, top+r); fmt.Print("│")
		move(left+w-1, top+r); fmt.Print("│")
	}
	move(left, top+h-1); fmt.Print("└")
	for i := 0; i < w-2; i++ { fmt.Print("─") }
	fmt.Print("┘")
}

func drawTextInBox(top, left, pad int, lines []string) {
	row := top + 1 + pad
	for _, s := range lines {
		move(left+1+pad, row)
		fmt.Print(s)
		row++
	}
}

func move(col, row int) {
	if col < 1 { col = 1 }
	if row < 1 { row = 1 }
	fmt.Printf("\x1b[%d;%dH", row, col)
}

func textSize(lines []string) (w, h int) {
	h = len(lines)
	for _, s := range lines {
		if ln := runeLen(s); ln > w { w = ln }
	}
	return
}

func runeLen(s string) (n int) { for range s { n++ }; return }

func termSizeOr(dw, dh int) (int, int) {
	if !term.IsTerminal(int(os.Stdout.Fd())) { return dw, dh }
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil { return dw, dh }
	// ensure non-zero to keep math sane on tiny terminals
	if w < 1 { w = 1 }
	if h < 1 { h = 1 }
	return w, h
}

func max(a, b int) int { if a > b { return a }; return b }

func approx(v, target, eps float64) bool { return math.Abs(v-target) <= eps }

func ansiColorRand() int {
	// normal + bright ANSI (reds..cyans)
	palette := []int{31, 32, 33, 34, 35, 36, 91, 92, 93, 94, 95, 96}
	return palette[rand.Intn(len(palette))]
}