package session
import (
	"container/list"
	"net/http"
	"sync"
	"time"
)

var memdep = &MemProvider{list: list.New(), sessions: make(map[string]*list.Element)}

// MemSessionStore memory session store.
// it saved sessions in a map in memory.
type MemSessionStore struct {
	sid          string 						     // session id
	timeAccessed time.Time		              // last access time
	value        map[interface{}]interface{} // session store
	lock sync.RWMutex
}

// Set value to memory session
func (st *MemSessionStore) Set(key, value interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.value[key] = value
	return nil
}

// Get value from memory session by key
func (st *MemSessionStore) Get(key interface{}) interface{} {
	st.lock.Lock()
	defer st.lock.Unlock()
	if v, ok := st.value[key]; ok {
		return v
	}
	return nil
}

// Delete in memory session by key
func (st *MemSessionStore) Delete(key interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	delete(st.value, key)
	return nil
}

// Flush clear all values in memory session
func (st *MemSessionStore) Flush() error {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.value = make(map[interface{}]interface{})
	return nil
}

// SessionID get this id of memory session store
func (st *MemSessionStore) SessionID() string {
	return st.sid
}

// SessionRelease Implement method, no used.
func (st *MemSessionStore) SesionRelease(w http.ResponseWriter) {
}
