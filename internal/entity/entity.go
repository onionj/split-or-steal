package entity

type DBModel interface {
	Table() string
	EntityID() ID
}
