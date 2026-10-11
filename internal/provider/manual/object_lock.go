package manual

import "sync"

// objectLocks serializes read-modify-write requests per NetBox object.
//
// NetBox merges a PATCH of custom_fields into the object's current values and
// saves the whole set, so two overlapping PATCHes of different fields on the
// same object lose one of the updates. Terraform applies independent
// netbox_custom_field_value resources in parallel, so every write holds the
// lock of its object for the duration of the request.
var objectLocks keyedMutex

// keyedMutex is a set of mutexes addressed by key. Entries are never removed:
// there is one per object path written during the lifetime of the provider
// process, which is bounded by the size of a single Terraform run.
type keyedMutex struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// lock blocks until the mutex for key is held and returns its unlock function.
func (k *keyedMutex) lock(key string) (unlock func()) {
	k.mu.Lock()
	if k.locks == nil {
		k.locks = map[string]*sync.Mutex{}
	}
	l, ok := k.locks[key]
	if !ok {
		l = &sync.Mutex{}
		k.locks[key] = l
	}
	k.mu.Unlock()
	l.Lock()
	return l.Unlock
}
