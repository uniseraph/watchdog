package watchdog


import (
	"sync"

	"github.com/docker/docker/api/types"
)

type Cache struct {
	entries map[string]*types.ContainerJSON
	sync.RWMutex
}

func NewContanerCache() *Cache {
	return &Cache{entries: make(map[string]*types.ContainerJSON)}
}

func (c *Cache) Add(s *types.ContainerJSON) {
	c.Lock()
	c.entries[s.ID] = s
	c.Unlock()
}

func (c *Cache) Remove(s *types.ContainerJSON) {
	c.Lock()
	delete(c.entries, s.ID)
	c.Unlock()
}

func (c *Cache) Reset(ss []*types.ContainerJSON) {
	c.Lock()
	c.entries = make(map[string]*types.ContainerJSON)
	if len(ss) > 0 {
		for _, s := range ss {
			c.entries[s.ID] = s
		}
	}
	c.Unlock()
}

func (c *Cache) Diff(ss []*types.ContainerJSON) ([]*types.ContainerJSON, []*types.ContainerJSON) {
	var (
		adds    []*types.ContainerJSON
		removes []*types.ContainerJSON
	)

	c.Lock()

	for _, s := range ss {
		if _, exists := c.entries[s.ID]; !exists {
			adds = append(adds, s)
		}
	}

	for id := range c.entries {
		found := false
		for _, s := range ss {
			if id == s.ID {
				found = true
				break
			}
		}
		if !found {
			removes = append(removes, c.entries[id])
		}
	}

	c.Unlock()

	return adds, removes
}

func (c *Cache) Get(id string) *types.ContainerJSON {
	c.RLock()
	defer c.RUnlock()
	return c.entries[id]
}
