package main

import "github.com/hajimehoshi/ebiten"

// Similar to ebiten/inpututil but it's querying only a limited number of keys.
// As each call to ebiten.IsKeyPressed is causing a sending messages over a channel
// to synchronize with the glfw thread, this way we save on cpu overhead of go
// runtime scheduling all those calls.
type KeyboardHandler struct {
	keysToHandle    []ebiten.Key
	keyPressed      []bool
	keyJustPressed  []bool
	keyJustReleased []bool
}

func NewKeyboardHandler() *KeyboardHandler {
	return &KeyboardHandler{
		keyPressed:      make([]bool, ebiten.KeyMax+1),
		keyJustPressed:  make([]bool, ebiten.KeyMax+1),
		keyJustReleased: make([]bool, ebiten.KeyMax+1)}
}
func (h *KeyboardHandler) AddKeyToHandle(key ebiten.Key) {
	h.keysToHandle = append(h.keysToHandle, key)
}
func (h *KeyboardHandler) Update() {
	for _, k := range h.keysToHandle {
		if ebiten.IsKeyPressed(k) {
			if !h.keyPressed[k] {
				h.keyPressed[k] = true
				h.keyJustPressed[k] = true
			} else {
				h.keyJustPressed[k] = false
			}
			h.keyJustReleased[k] = false
		} else {
			if h.keyPressed[k] {
				h.keyPressed[k] = false
				h.keyJustReleased[k] = true
			} else {
				h.keyJustReleased[k] = false
			}
			h.keyJustPressed[k] = false
		}
	}
}

func (h *KeyboardHandler) IsKeyJustPressed(key ebiten.Key) bool {
	return h.keyJustPressed[key]
}
func (h *KeyboardHandler) IsKeyJustReleased(key ebiten.Key) bool {
	return h.keyJustReleased[key]
}
