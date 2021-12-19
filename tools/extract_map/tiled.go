package main

import (
	"encoding/json"
	"fmt"
)

type Orientation int

var Orthogonal Orientation = 0
var Isometric Orientation = 1
var Staggered Orientation = 2
var Hexagonal Orientation = 3

func (o Orientation) MarshalText() ([]byte, error) {
	switch o {
	case Orthogonal:
		return []byte("orthogonal"), nil
	case Isometric:
		return []byte("isometric"), nil
	case Staggered:
		return []byte("staggered"), nil
	case Hexagonal:
		return []byte("hexagonal"), nil
	default:
		return nil, fmt.Errorf("unknown orientation: %d", o)
	}
}
func (o *Orientation) UnmarshalText(text []byte) error {
	var s string
	if err := json.Unmarshal(text, &s); err != nil {
		return err
	}
	var err error
	switch s {
	case "orthogonal":
		*o = Orthogonal
	case "isometric":
		*o = Isometric
	case "staggered":
		*o = Staggered
	case "hexagonal":
		*o = Hexagonal
	default:
		err = fmt.Errorf("unknown orientation: \"%s\"", s)
	}
	return err
}

type RenderOrder int

var RightDown RenderOrder = 0
var RightUp RenderOrder = 1
var LeftDown RenderOrder = 2
var LeftUp RenderOrder = 3

func (ro RenderOrder) MarshalText() ([]byte, error) {
	switch ro {
	case RightDown:
		return []byte("right-down"), nil
	case RightUp:
		return []byte("right-up"), nil
	case LeftDown:
		return []byte("left-down"), nil
	case LeftUp:
		return []byte("left-up"), nil
	}
	return nil, fmt.Errorf("unknown renderorder: %d", ro)
}
func (ro *RenderOrder) UnmarshalText(text []byte) error {
	var s string
	if err := json.Unmarshal(text, &s); err != nil {
		return err
	}
	var err error
	switch s {
	case "right-down":
		*ro = RightDown
	case "right-up":
		*ro = RightUp
	case "left-down":
		*ro = LeftDown
	case "left-up":
		*ro = LeftUp
	default:
		err = fmt.Errorf("unknown renderorder: \"%s\"", s)
	}
	return err
}

type Axis int

var X Axis = 0
var Y Axis = 1

func (a Axis) MarshalText() ([]byte, error) {
	switch a {
	case X:
		return []byte("x"), nil
	case Y:
		return []byte("y"), nil
	}
	return nil, fmt.Errorf("unknown axis: %d", a)
}
func (a *Axis) UnmarshalText(text []byte) error {
	var s string
	if err := json.Unmarshal(text, &s); err != nil {
		return err
	}
	var err error
	switch s {
	case "x":
		*a = X
	case "y":
		*a = Y
	default:
		err = fmt.Errorf("unknown axis: \"%s\"", s)
	}
	return err
}

type StaggerIndex int

var Odd StaggerIndex = 0
var Even StaggerIndex = 1

func (si StaggerIndex) MarshalText() ([]byte, error) {
	switch si {
	case Odd:
		return []byte("odd"), nil
	case Even:
		return []byte("even"), nil
	}
	return nil, fmt.Errorf("unknown staggeraxis: %d", si)
}
func (si *StaggerIndex) UnmarshalText(text []byte) error {
	var s string
	if err := json.Unmarshal(text, &s); err != nil {
		return err
	}
	var err error
	switch s {
	case "odd":
		*si = Odd
	case "even":
		*si = Even
	default:
		err = fmt.Errorf("unknown staggeraxis: \"%s\"", s)
	}
	return err
}

type MapType int

var Map MapType = 0

func (mt MapType) MarshalText() ([]byte, error) {
	switch mt {
	case Map:
		return []byte("map"), nil
	}
	return nil, fmt.Errorf("unknown map type: %d", mt)
}
func (mt *MapType) UnmarshalText(text []byte) error {
	var s string
	if err := json.Unmarshal(text, &s); err != nil {
		return err
	}
	var err error
	switch s {
	case "map":
		*mt = Map
	default:
		err = fmt.Errorf("unknown map type: \"%s\"", s)
	}
	return err
}

type LayerType int

var TileLayer LayerType = 0
var ObjectGroup LayerType = 1
var ImageLayer LayerType = 2
var Group LayerType = 3

func (lt LayerType) MarshalText() ([]byte, error) {
	switch lt {
	case TileLayer:
		return []byte("tilelayer"), nil
	case ObjectGroup:
		return []byte("objectgroup"), nil
	case ImageLayer:
		return []byte("imagelayer"), nil
	case Group:
		return []byte("group"), nil
	}
	return nil, fmt.Errorf("unknown layer type: %d", lt)
}
func (lt *LayerType) UnmarshalText(text []byte) error {
	var s string
	if err := json.Unmarshal(text, &s); err != nil {
		return err
	}
	var err error
	switch s {
	case "tilelayer":
		*lt = TileLayer
	case "objectgroup":
		*lt = ObjectGroup
	case "imagelayer":
		*lt = ImageLayer
	case "group":
		*lt = Group
	default:
		err = fmt.Errorf("unknown layer type: \"%s\"", s)
	}
	return err
}

type Layer struct {
	Data    []int     `json:"data"`
	Height  int       `json:"height"`
	Width   int       `json:"width"`
	ID      int       `json:"id"`
	Name    string    `json:"name"`
	Opacity float64   `json:"opacity"`
	Type    LayerType `json:"type"`
	Visible bool      `json:"visible"`
	X       int       `json:"x"`
	Y       int       `json:"y"`
}

type GridOrientation int

var OrthogonalGrid GridOrientation = 0
var IsometricGrid GridOrientation = 1

func (o GridOrientation) MarshalText() ([]byte, error) {
	switch o {
	case OrthogonalGrid:
		return []byte("orthogonal"), nil
	case IsometricGrid:
		return []byte("isometric"), nil
	default:
		return nil, fmt.Errorf("unknown grid orientation: %d", o)
	}
}
func (o *GridOrientation) UnmarshalText(text []byte) error {
	var s string
	if err := json.Unmarshal(text, &s); err != nil {
		return err
	}
	var err error
	switch s {
	case "orthogonal":
		*o = OrthogonalGrid
	case "isometric":
		*o = IsometricGrid
	default:
		err = fmt.Errorf("unknown grid orientation: \"%s\"", s)
	}
	return err
}

type Grid struct {
	Height      int             `json:"height"`
	Width       int             `json:"width"`
	Orientation GridOrientation `json:"orientation"`
}

type Tile struct {
	ID          int    `json:"id"`
	Image       string `json:"string"`
	ImageHeight int    `json:"imageheight"`
	ImageWidth  int    `json:"imagewidth"`
}

type TileSet struct {
	Columns     int    `json:"columns"`
	FirstGID    int    `json:"firstgid"`
	Grid        Grid   `json:"grid"`
	Margin      int    `json:"margin"`
	Name        string `json:"name"`
	Spacing     int    `json:"spacing"`
	TileCount   int    `json:"tilecount"`
	TileHeight  int    `json:"tileheight"`
	TileWidth   int    `json:"tilewidth"`
	Tiles       []Tile `json:"tiles"`
	Image       string `json:"image"`
	ImageHeight int    `json:"imageheight"`
	ImageWidth  int    `json:"imagewidth"`
}

type TiledMap struct {
	Height        int          `json:"height"`
	Width         int          `json:"width"`
	Infinite      bool         `json:"infinite"`
	Layers        []Layer      `json:"layers"`
	NextLayerID   int          `json:"nextlayerid"`
	NextObjectID  int          `json:"nextobjectid"`
	Orientation   Orientation  `json:"orientation"`
	RenderOrder   RenderOrder  `json:"renderorder"`
	StaggerAxis   Axis         `json:"staggeraxis"`
	StaggerIndex  StaggerIndex `json:"staggerindex"`
	HexSideLength int          `json:"hexsidelength"`
	TileHeight    int          `json:"tileheight"`
	TileWidth     int          `json:"tilewidth"`
	Type          MapType      `json:"type"`
	TileSets      []TileSet    `json:"tilesets"`
}
