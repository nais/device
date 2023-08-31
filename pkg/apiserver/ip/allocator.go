package ip

type Allocator interface {
	NextIP([]string) (string, error)
}
