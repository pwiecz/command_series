package main

import "github.com/hajimehoshi/ebiten"

type MouseHandler struct {
	buttonsToHandle    []ebiten.MouseButton
	buttonPressed      map[ebiten.MouseButton]bool
	buttonJustPressed  map[ebiten.MouseButton]bool
	buttonJustReleased map[ebiten.MouseButton]bool
}

func NewMouseHandler() *MouseHandler {
	return &MouseHandler{
		buttonPressed:      make(map[ebiten.MouseButton]bool),
		buttonJustPressed:  make(map[ebiten.MouseButton]bool),
		buttonJustReleased: make(map[ebiten.MouseButton]bool)}
}

func (h *MouseHandler) AddButtonToHandle(button ebiten.MouseButton) {
	h.buttonsToHandle = append(h.buttonsToHandle, button)
}

func (h *MouseHandler) Update() {
	for _, b := range h.buttonsToHandle {
		if ebiten.IsMouseButtonPressed(b) {
			if !h.buttonPressed[b] {
				h.buttonPressed[b] = true
				h.buttonJustPressed[b] = true
			} else {
				h.buttonJustPressed[b] = false
			}
			h.buttonJustReleased[b] = false
		} else {
			if h.buttonPressed[b] {
				h.buttonPressed[b] = false
				h.buttonJustReleased[b] = true
			} else {
				h.buttonJustReleased[b] = false
			}
			h.buttonJustPressed[b] = false
		}
	}
}

func (h *MouseHandler) IsButtonJustPressed(button ebiten.MouseButton) bool {
	return h.buttonJustPressed[button]
}
func (h *MouseHandler) IsButtonJustReleased(button ebiten.MouseButton) bool {
	return h.buttonJustReleased[button]
}
