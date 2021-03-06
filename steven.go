// Copyright 2015 Matthew Collins
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/thinkofdeath/steven/platform"
	"github.com/thinkofdeath/steven/protocol/mojang"
	"github.com/thinkofdeath/steven/render"
)

var loadChan = make(chan struct{})
var debug bool

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	if len(os.Args) == 0 {
		fmt.Println("steven must be run via the mojang launcher")
		return
	}

	// Can't use flags as we need to support a weird flag
	// format
	var username, uuid, accessToken, server string

	for i, arg := range os.Args {
		switch arg {
		case "--username":
			username = os.Args[i+1]
		case "--uuid":
			uuid = os.Args[i+1]
		case "--accessToken":
			accessToken = os.Args[i+1]
		case "--server":
			server = os.Args[i+1]
		case "--debug":
			debug = true
		}
	}

	// Start connecting whilst starting the renderer
	go startConnection(mojang.Profile{
		Username:    username,
		ID:          uuid,
		AccessToken: accessToken,
	}, server)

	go func() {
		render.LoadTextures()
		initBlocks()
		loadChan <- struct{}{}
	}()

	platform.Init(platform.Handler{
		Start:  start,
		Draw:   draw,
		Move:   move,
		Rotate: rotate,
		Action: action,
	})
}

func start() {
	<-loadChan
	render.Start(debug)
}

func rotate(x, y float64) {
	Client.Yaw -= x
	Client.Pitch -= y
}

var mf, ms float64

func move(f, s float64) {
	mf, ms = f, s
}

func action(action platform.Action) {
	switch action {
	case platform.Debug:
	case platform.JumpToggle:
		Client.Jumping = !Client.Jumping
	}
}

var maxBuilders = runtime.NumCPU() * 2

var (
	ready            bool
	freeBuilders     = maxBuilders
	completeBuilders = make(chan buildPos, maxBuilders)
	syncChan         = make(chan func(), 200)
	ticker           = time.NewTicker(time.Second / 20)
	lastFrame        = time.Now()
)

func draw() {
	now := time.Now()
	diff := now.Sub(lastFrame)
	lastFrame = now
	delta := float64(diff.Nanoseconds()) / (float64(time.Second) / 60)
	delta = math.Min(math.Max(delta, 0.3), 1.6)
handle:
	for {
		select {
		case err := <-errorChan:
			panic(err)
		case packet := <-readChan:
			defaultHandler.Handle(packet)
		case pos := <-completeBuilders:
			c := chunkMap[chunkPosition{pos.X, pos.Z}]
			freeBuilders++
			if c != nil {
				s := c.Sections[pos.Y]
				if s != nil {
					s.building = false
				}
			}
		case f := <-syncChan:
			f()
		default:
			break handle
		}
	}

	if ready {
		Client.renderTick(delta)
		select {
		case <-ticker.C:
			tick()
		default:
		}
	}

	render.Draw(delta)
	chunks := sortedChunks()

	// Search for 'dirty' chunk sections and start building
	// them if we have any builders free. To prevent race conditions
	// two flags are used, dirty and building, to allow a second
	// build to be requested whilst the chunk is still building
	// without either losing the change or having two builds
	// for the same section going on at once (where the second
	// could finish quicker causing the old version to be
	// displayed.
dirtyClean:
	for _, c := range chunks {
		for _, s := range c.Sections {
			if s == nil {
				continue
			}
			if freeBuilders <= 0 {
				break dirtyClean
			}
			if s.dirty && !s.building {
				freeBuilders--
				s.dirty = false
				s.building = true
				s.build(completeBuilders)
			}
		}
	}
}

// tick is called 20 times a second (bar any preformance issues).
// Minecraft is built around this fact so we have to follow it
// as well.
func tick() {
	Client.tick()
}
