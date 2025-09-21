// dvd_edge_keys_color.go
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

const (
	fps       = 60
	boxW      = 10
	boxH      = 5
	clearHome = "\x1b[2J\x1b[H"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Enter alt screen & hide cursor
	fmt.Print("\x1b[?1049h\x1b[?25l")
	defer fmt.Print("\x1b[0m\x1b[2J\x1b[H\x1b[?25h\x1b[?1049l")

	// Raw mode for keypress exit
	oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Context cancels on Ctrl-C or any key
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	keyCh := make(chan struct{}, 1)
	go func() {
		buf := make([]byte, 1)
		// any byte -> exit
		for {
			_, err := os.Stdin.Read(buf)
			if err == nil {
				select { case keyCh <- struct{}{}: default: }
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Arena: origin in [0..aw], [0..ah] for top-left of box
	W, H := termSizeOr(120, 30)
	aw, ah := max(0, W-boxW), max(0, H-boxH)

	// Pos/vel
	x := rand.Float64() * float64(aw)
	y := rand.Float64() * float64(ah)
	speed := 35.0
	angle := 0.82
	vx := speed * math.Cos(angle)
	vy := speed * math.Sin(angle)

	// Color state (31..36)
	color := 31 + rand.Intn(6)

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
			// Resize
			w2, h2 := termSizeOr(120, 30)
			if w2 != W || h2 != H {
				W, H = w2, h2
				aw, ah = max(0, W-boxW), max(0, H-boxH)
				if x > float64(aw) {
					x = float64(aw)
				}
				if y > float64(ah) {
					y = float64(ah)
				}
			}

			// Timing
			now := time.Now()
			dt := now.Sub(last).Seconds()
			if dt <= 0 {
				dt = 1.0 / fps
			}
			last = now

			// Predict
			nx := x + vx*dt
			ny := y + vy*dt

			// Continuous collision timing to detect exact corner
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
			if tx >= 0 && tx <= 1 && ty >= 0 && ty <= 1 && math.Abs(tx-ty) < 1e-3 {
				// exact corner within the frame
				corner = true
				// move to corner impact
				x += vx * dt * tx
				y += vy * dt * ty
				// bounce both
				vx, vy = -vx, -vy
				// finish remaining subframe
				rem := (1 - ((tx + ty) / 2)) * dt
				x += vx * rem
				y += vy * rem
			} else {
				// Normal per-axis reflections
				if nx < 0 {
					nx = -nx
					vx = -vx
				} else if nx > float64(aw) {
					nx = 2*float64(aw) - nx
					vx = -vx
				}
				if ny < 0 {
					ny = -ny
					vy = -vy
				} else if ny > float64(ah) {
					ny = 2*float64(ah) - ny
					vy = -vy
				}
				x, y = nx, ny
			}

			// On exact-corner, change to a different color
			if corner {
				newC := color
				for newC == color {
					newC = 31 + rand.Intn(6) // 31..36
				}
				color = newC
			}

			// Render
			fmt.Print(clearHome)
			fmt.Printf("\x1b[%dm", color)
			ix := int(math.Round(x)) + 1 // 1-based screen coords
			iy := int(math.Round(y)) + 1
			drawDVDBox(ix, iy)
			fmt.Print("\x1b[0m")
		}
	}
}

func termSizeOr(dw, dh int) (int, int) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return dw, dh
	}
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return dw, dh
	}
	if w < boxW {
		w = boxW
	}
	if h < boxH {
		h = boxH
	}
	return w, h
}

func drawDVDBox(x, y int) {
	lines := []string{
		"┌────────┐",
		"│        │",
		"│  DVD   │",
		"│        │",
		"└────────┘",
	}
	for i, s := range lines {
		moveCursor(x, y+i)
		fmt.Print(s)
	}
}

func moveCursor(col, row int) { // 1-based
	if col < 1 {
		col = 1
	}
	if row < 1 {
		row = 1
	}
	fmt.Printf("\x1b[%d;%dH", row, col)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}