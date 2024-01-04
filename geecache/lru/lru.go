package lru

import (
	"container/list"
)

// Cache is an LRU cache. It is not safe for concurrent access.
type Cache struct {
	maxBytes int64
	nbytes   int64
	ll       *list.List
	cache    map[string]*list.Element
	// 某条记录被移除时的回调函数，可以为 nil
	OnEvicted func(key string, value Value)
}

// 双向链表节点的数据类型
type entry struct {
	// 淘汰节点时方便从字典中删除 key
	key   string
	value Value
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get looks up a key's value
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		// 双向链表作为队列，队首队尾是相对的，在这里约定 front 为队尾
		c.ll.MoveToFront(ele)
		// 类型断言
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// RemoveOldest removes the oldest item
func (c *Cache) RemoveOldest() {
	// 移除最近最少访问的节点
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		// 删除映射关系
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add adds a value to the cache
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}
