package wipe

// destroyableResource is an object that needs to be wiped
type destroyableResource interface {
	String() string
	Destroy() error
	Priority() int
}

// destroyableResources is a priority-ordered collection of destroyable resources
type destroyableResources []destroyableResource

// make destroyableResources sortable by priority
func (a destroyableResources) Len() int      { return len(a) }
func (a destroyableResources) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a destroyableResources) Less(i, j int) bool {
	p1, p2 := a[i].Priority(), a[j].Priority()
	if p1 == p2 {
		return a[i].String() < a[j].String()
	}
	return p1 < p2
}
