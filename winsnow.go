package main

import (
	"image/color"
	"log"
	"math/rand"
	"syscall"
	"time"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/sys/windows"
)

// Constants for window positioning
const (
	HWND_BOTTOM      = 1
	HWND_TOPMOST     = -1
	HWND_NOTOPMOST   = -2
	SWP_NOMOVE       = 0x0002
	SWP_NOSIZE       = 0x0001
	SWP_NOACTIVATE   = 0x0010
	SWP_SHOWWINDOW   = 0x0040
	GWL_EXSTYLE      = -20
	WS_EX_LAYERED    = 0x80000
	WS_EX_NOACTIVATE = 0x08000000
)

const (
	screenWidth   = 1920 // Default, will be set to actual screen size
	screenHeight  = 1080 // Default, will be set to actual screen size
	numSnowflakes = 300
)

// Snowflake represents a single snow particle
type Snowflake struct {
	x, y      float64
	size      float64
	speed     float64
	drift     float64
	windSpeed float64
}

// Game implements ebiten.Game interface
type Game struct {
	snowflakes     []Snowflake
	screenWidth    int
	screenHeight   int
	wind           float64 // Current wind strength
	windTarget     float64 // Target wind strength
	windChangeTime float64 // Time until next wind change
}

// Initialize creates all the snowflakes
func (g *Game) Initialize() {
	// Get the primary monitor size
	g.screenWidth, g.screenHeight = ebiten.ScreenSizeInFullscreen()

	// Initialize wind
	g.wind = 0
	g.windTarget = 0
	g.windChangeTime = 0

	// Create snowflakes
	g.snowflakes = make([]Snowflake, numSnowflakes)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := range g.snowflakes {
		g.snowflakes[i] = Snowflake{
			x:     r.Float64() * float64(g.screenWidth),
			y:     r.Float64() * float64(g.screenHeight),
			size:  1.0 + r.Float64()*3.0,
			speed: 6.0 + r.Float64()*10.0, // Min: 6.0, Max: 16.0
			drift: 0,
		}
	}
}

// Update updates the game state (implementing ebiten.Game)
func (g *Game) Update() error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Update wind
	g.windChangeTime -= 1.0
	if g.windChangeTime <= 0 {
		// Set new wind target
		g.windTarget = (r.Float64()*2 - 1.0) * 0.8 // Range: -0.8 to 0.8
		g.windChangeTime = 60 + r.Float64()*120    // Change every 60-180 frames
	}

	// Gradually adjust wind toward target (subtle change)
	g.wind = g.wind*0.99 + g.windTarget*0.01

	// Update snowflakes
	for i := range g.snowflakes {
		// Apply wind effect - larger flakes affected less by wind
		windEffect := g.wind / g.snowflakes[i].size
		g.snowflakes[i].x += windEffect

		// Apply velocity
		g.snowflakes[i].y += g.snowflakes[i].speed

		// Reset if out of bounds
		if g.snowflakes[i].y > float64(g.screenHeight) {
			g.snowflakes[i].y = 0
			g.snowflakes[i].x = r.Float64() * float64(g.screenWidth)
		}

		// Wrap around left/right edges if needed
		if g.snowflakes[i].x < 0 {
			g.snowflakes[i].x = float64(g.screenWidth)
		} else if g.snowflakes[i].x > float64(g.screenWidth) {
			g.snowflakes[i].x = 0
		}
	}

	return nil
}

// Draw draws the game screen (implementing ebiten.Game)
func (g *Game) Draw(screen *ebiten.Image) {
	// Clear the screen with transparent black
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// Draw snowflakes
	for _, flake := range g.snowflakes {
		size := int(flake.size)
		x, y := int(flake.x), int(flake.y)

		if size <= 1 {
			screen.Set(x, y, color.White)
		} else {
			// Draw larger snowflakes as circles
			for dx := -size / 2; dx <= size/2; dx++ {
				for dy := -size / 2; dy <= size/2; dy++ {
					if dx*dx+dy*dy <= size*size/4 {
						screen.Set(x+dx, y+dy, color.White)
					}
				}
			}
		}
	}
}

// Layout returns the screen dimensions (implementing ebiten.Game)
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenWidth, g.screenHeight
}

// SetWindowToBottom sets the window to be behind all applications but in front of the desktop
func SetWindowToBottom() {
	// Get the window handle using Windows API
	user32 := windows.NewLazySystemDLL("user32.dll")
	procFindWindow := user32.NewProc("FindWindowW")
	procSetWindowPos := user32.NewProc("SetWindowPos")
	procGetForegroundWindow := user32.NewProc("GetForegroundWindow")

	// Convert window title to UTF16
	title, _ := syscall.UTF16PtrFromString("Snow Wallpaper")

	// Find the window by title
	hwnd, _, _ := procFindWindow.Call(
		0,
		uintptr(unsafe.Pointer(title)),
	)

	if hwnd == 0 {
		// Try with class name instead
		className, _ := syscall.UTF16PtrFromString("Ebiten")
		hwnd, _, _ = procFindWindow.Call(
			uintptr(unsafe.Pointer(className)),
			0,
		)

		if hwnd == 0 {
			log.Println("Could not find window handle, will retry later")
			return
		}
	}

	// Get the foreground window
	fgHwnd, _, _ := procGetForegroundWindow.Call()

	// Set the window position to be at the bottom of the Z-order
	// and make sure it's not activated
	procSetWindowPos.Call(
		hwnd,
		uintptr(HWND_BOTTOM),
		0, 0, 0, 0,
		uintptr(SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE|SWP_SHOWWINDOW),
	)

	// Restore focus to the previous foreground window
	if fgHwnd != 0 && fgHwnd != hwnd {
		procSetWindowPos.Call(
			fgHwnd,
			0, // Just behind HWND_TOP
			0, 0, 0, 0,
			uintptr(SWP_NOMOVE|SWP_NOSIZE|SWP_SHOWWINDOW),
		)
	}
}

func main() {
	// Create game instance
	game := &Game{}
	game.Initialize()

	// Configure Ebiten
	ebiten.SetWindowTitle("Snow Wallpaper")
	ebiten.SetWindowSize(game.screenWidth, game.screenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(true)
	ebiten.SetWindowDecorated(false) // No window decorations (title bar, etc.)
	ebiten.SetWindowPosition(0, 0)   // Position window at top-left corner
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetScreenTransparent(true)

	// Run window positioning in background repeatedly
	go func() {
		// Give the window time to be created first
		time.Sleep(500 * time.Millisecond)

		// Try positioning the window repeatedly
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			SetWindowToBottom()
		}
	}()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
