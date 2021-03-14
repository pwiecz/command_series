package lib

type UnitCoords struct {
	X, Y int
}

type MapCoords struct {
	X, Y int
}

func (c UnitCoords) ToMapCoords() MapCoords {
	if c.X >= 0 {
		return MapCoords{c.X / 2, c.Y}
	}
	return MapCoords{(c.X - 1) / 2, c.Y}
}
func (c MapCoords) ToUnitCoords() UnitCoords {
	return UnitCoords{c.X*2 + Abs(c.Y)%2, c.Y}
}
