package main

import "github.com/hajimehoshi/ebiten"

type KeyboardHandler struct {
	keyPressed []bool
	keyJustPressed []bool
}

func NewKeyboardHandler() *KeyboardHandler {
	return &KeyboardHandler {
		keyPressed: make([]bool, ebiten.KeyMax+1),
		keyJustPressed: make([]bool, ebiten.KeyMax+1),
	}
}
func (h *KeyboardHandler) Update() {
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		if ebiten.IsKeyPressed(k) {
			if !h.keyPressed[k] {
				h.keyPressed[k] = true
				h.keyJustPressed[k] = true
			} else {
				h.keyJustPressed[k] = false
			}
		} else {
			h.keyPressed[k] = false
			h.keyJustPressed[k] = false
		}
	}
}

func (h *KeyboardHandler) IsKeyJustPressed(key ebiten.Key) bool {
	return h.keyJustPressed[key]
}
