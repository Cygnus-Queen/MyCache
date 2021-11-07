package geeCache

//ByteView 缓存值的数据类型
type ByteView struct {
	b []byte
}

func (c ByteView) Len() int{
	return len(c.b)
}

func (c ByteView) String() string{
	return string(c.b)
}

func (c ByteView) ByteSlice() []byte{
	return cloneByte(c.b)
}

func cloneByte(b []byte) []byte{
	c := make([]byte,len(b))
	copy(c,b)
	return c
}

