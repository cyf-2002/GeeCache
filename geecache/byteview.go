package geecache

// 抽象一个只读数据结构，表示缓存值
type ByteView struct {
	b []byte
}

// 实现 Value 接口
func (v ByteView) Len() int {
	return len(v.b)
}

// return a copy of the data as a byte slice
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// return the data as a string
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
