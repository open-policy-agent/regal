//go:build regal_enable_novelty

package novelty

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	sprite "github.com/pdevine/go-asciisprite"
	tm "github.com/pdevine/go-asciisprite/termbox"
)

func HappyHolidays() error {
	var allSprites sprite.SpriteGroup

	flakes := []string{flake1, flake2, flake3, flake4, flake5, flake6, flake7}

	if err := tm.Init(); err != nil {
		return fmt.Errorf("termbox init failed: %w", err)
	}
	defer tm.Close()

	width, height := tm.Size()

	eventQueue := make(chan tm.Event)

	go func() {
		for {
			eventQueue <- tm.PollEvent()
		}
	}()

	done := make(chan bool, 1)

	messageSprite := newSpriteInstance(
		width/2,
		height/2-len(strings.Split(messageCostume, "\n"))/2-5,
		false,
		messageCostume,
	)
	allSprites.Sprites = append(allSprites.Sprites, messageSprite)

	for range 30 {
		//nolint:gosec
		flakeX := rand.Intn(width)
		//nolint:gosec
		flakey := rand.Intn(height*2) - height*2
		//nolint:gosec
		costume := flakes[rand.Intn(len(flakes))]
		f := newSpriteInstance(flakeX, flakey, true, costume)
		allSprites.Sprites = append(allSprites.Sprites, f)
	}

	go func() {
		time.Sleep(5 * time.Second)
		done <- true
	}()

	for {
		tm.Clear(tm.ColorDefault, tm.ColorDefault)

		select {
		case ev := <-eventQueue:
			if ev.Type == tm.EventKey {
				if ev.Key == tm.KeyCtrlC || ev.Key == tm.KeyCtrlD || ev.Key == tm.KeyEsc {
					return nil
				}
			}
		case <-done:
			return nil
		default:
			allSprites.Update()
			allSprites.Render()
			time.Sleep(70 * time.Millisecond)
		}
	}
}

const flake1 = `
 __/\__
 \_\/_/
 /_/\_\
   \/
`

const flake2 = `
  _\/\/_
 _\_\/_/_
  /_/\_\
   /\/\
`

const flake3 = `
     \/
 _\_\/\/_/_
  _\_\/_/_
 __/_/\_\__
  / /\/\ \
     /\
`

const flake4 = `
      /\
 __   \/   __
 \_\_\/\/_/_/
   _\_\/_/_
  __/_/\_\__
 /_/ /\/\ \_\
      /\
      \/
`

const flake5 = `
  __/  \__
   _\/\/_
 \_\_\/_/_/
 / /_/\_\ \
  __/\/\__
    \  /
`

const flake6 = `
    __  __
   /_/  \_\
    _\/\/_
 /\_\_\/_/_/\
 \/ /_/\_\ \/
   __/\/\__
   \_\  /_/
`

const flake7 = `
     \__  __/
     /_/  \_\
      _\/\/_
 __/\_\_\/_/_/\__
   \/ /_/\_\ \/
     __/\/\__
     \_\  /_/
     /      \
`

//nolint:dupword
const messageCostume = `
                No Violations Reported!

     A                     A                    A
    A A                 AA  AA                 A AA
   AA AA            AAAAA    AAAAA            AA AA
   AA  AA        AAAAAAAA    AA     A        AA  AA
    A    AAAAAAAAAAAAAA A    AA      AAAAAAAA    AA
    AA    AAAAAAAAAAA   A    AA       AAAAAAA   AA
      AA AAAAAAAAAAA    A    AA         AAAAA AAA
        AAAAAAAAAAAA    A    AA         AAAAAAA
         AAAAAAAAAAAAAAAA    AAAAAAAAAAAAAAAA
          AA            A    AA           AA
           AA           A    AA           AA
         AA AAAAAAAAAAAAAA   AAAAAAAAAAAAAAAAA
        A  AA  A    AA    AA     A    AA AA  A
        A  A            AAAAAA       AA   AA AA
        AAA      AAAAAA        AAAAAA      AAA
         AA              AA AA              AA
         AA                                 A
           AAA                           AA
           AAA                           AAA
         AA  AAAAA                  AAAAA  AAA
        AA        AAAAA        AAAAAA        A
        A         AA   AAAAAAAA    AA        AA
        A       AAAA    AA  AA     AAA       AA
        A     AAAAAA   AAA  AAAA  AAAAAA     AA
        A AAAAAAAAAAA  AAA  AAAA  AAAAAAAAAA AA
          AA  AAAAAAAA   AA AA   AAAAAAA  AA
          AA   AAAAAAA     A    AAAAAAAA   A
           AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA

               Happy Holidays from the Regal Authors!
`

type spriteInstance struct {
	sprite.BaseSprite
	VX int
	VY int
}

func newSpriteInstance(x, y int, randVY bool, costume string) *spriteInstance {
	vy := 0
	if randVY {
		//nolint:gosec
		vy = rand.Intn(2) + 2
	}

	width := 0
	lines := strings.Split(messageCostume, "\n")

	for _, line := range lines {
		if len(line) > width {
			width = len(line)
		}
	}

	si := &spriteInstance{
		BaseSprite: sprite.BaseSprite{
			X:       x - width/2,
			Y:       y,
			Visible: true,
		},
		VY: vy,
	}

	si.AddCostume(sprite.NewCostume(costume, ' '))

	return si
}

func (f *spriteInstance) Update() {
	f.X += f.VX
	f.Y += f.VY
}
