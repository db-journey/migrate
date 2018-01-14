// Package direction just holds convenience constants for Up and Down migrations.
package direction

// Direction of migration
type Direction int

// Directions
const (
	Up   Direction = +1
	Down           = -1
)

func (d Direction) String() string {
	switch d {
	case Up:
		return "up"
	case Down:
		return "down"
	}
	panic("invalid direction")
}
