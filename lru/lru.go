package lru

import (
	"container/list"
	"fmt"
)

//Value 近似空接口，需要实现返回其自身所占空间大小的函数
type Value interface {
	Len() int
}

//entry 链表的数据类型，键值对
type entry struct {
	key string
	value Value
}

// Cache 一个使用lru策略的告诉缓存结构
type Cache struct {
	maxBytes int64 //缓存最大容量
	nBytes int64   //缓存目前占用的容量
	ll *list.List  //存储键值对链表
	cache map[string]*list.Element
	onEvicted func(key string, value Value)
}

//New 封装，创建一个Cache
func New(maxBytes int64, onEvicted func(string,Value)) *Cache {
	temp := new(Cache)
	temp.maxBytes = maxBytes
	temp.nBytes = 0
	temp.ll = new(list.List)
	temp.cache = make(map[string]*list.Element)
	temp.onEvicted = onEvicted
	return temp
}

func (c *Cache) Get(key string) (value Value,ok bool){
	if ele,ok := c.cache[key]; ok{
		c.ll.MoveToFront(ele)
		temp := ele.Value.(*entry)
		return temp.value,true
	}
	return
}

func (c *Cache) RemoveOldest(){
	ele := c.ll.Back()
	if ele != nil{
		c.ll.Remove(ele)
		v := ele.Value.(*entry)
		delete(c.cache,v.key)
		c.nBytes -= int64(v.value.Len())+ int64(len(v.key))
		if c.onEvicted != nil{
			c.onEvicted(v.key,v.value)
		}
	}
 	fmt.Println("Cache is empty")
}

//Add 添加新元素，若元素已存在则变为修改，若容量超出则移出部分节点
func (c *Cache) Add(key string, value Value){
	if ele,ok := c.cache[key];ok{
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	}else {
		kv := new(entry)
		kv.value = value
		kv.key = key
		ele := c.ll.PushFront(kv)
		c.cache[key] = ele
		c.nBytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.nBytes > c.maxBytes{
		c.RemoveOldest()
	}
}

//Len 返回容器内部存储节点的个数
func (c *Cache) Len() int{
	return c.ll.Len()
}