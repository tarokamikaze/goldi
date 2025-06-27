package goldi

// A StringSet represents a set of strings.
// Uses empty struct{} as value to minimize memory usage.
type StringSet map[string]struct{}

// NewStringSet creates a new StringSet with optional initial capacity
func NewStringSet(capacity ...int) StringSet {
	if len(capacity) > 0 && capacity[0] > 0 {
		return make(StringSet, capacity[0])
	}
	return make(StringSet)
}

// Set adds a value to the set.
func (s StringSet) Set(value string) {
	s[value] = struct{}{}
}

// Contains returns true if the given value is contained in this string set.
func (s StringSet) Contains(value string) bool {
	_, exists := s[value]
	return exists
}

// Remove removes a value from the set.
func (s StringSet) Remove(value string) {
	delete(s, value)
}

// Len returns the number of elements in the set.
func (s StringSet) Len() int {
	return len(s)
}

// Clear removes all elements from the set.
func (s StringSet) Clear() {
	for k := range s {
		delete(s, k)
	}
}

// ToSlice returns all values as a slice.
func (s StringSet) ToSlice() []string {
	if len(s) == 0 {
		return nil
	}

	// Use memory pool for slice allocation
	pool := GetGlobalMemoryPool()
	result := pool.GetStringSlice()
	defer pool.PutStringSlice(result)

	// Ensure capacity
	if cap(result) < len(s) {
		result = make([]string, 0, len(s))
	} else {
		result = result[:0]
	}

	for k := range s {
		result = append(result, k)
	}

	// Return a copy to avoid pool interference
	finalResult := make([]string, len(result))
	copy(finalResult, result)
	return finalResult
}
