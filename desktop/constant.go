package main

type Environment int

const (
	Development Environment = iota
	Production
)

func (e Environment) String() string {
	return [...]string{"development", "production"}[e]
}

type Platform int

const (
	Windows Platform = iota
	Darwin
	Linux
)

func (p Platform) String() string {
	return [...]string{"windows", "darwin", "linux"}[p]
}
