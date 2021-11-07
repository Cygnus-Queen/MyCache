package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func ([]byte) uint32

type Map struct {
	hash Hash               //从key到所需存储的虚拟节点的映射,是一个方法
	keys []int				//虚拟节点构成的环
	hashMap	map[int]string //从虚拟节点到真实节点的映射
	replicas int			//一个真实节点对应的虚拟节点数量
}

func NewMap(NumberOfReplicas int,h Hash) *Map{
	m := new(Map)
	m.replicas = NumberOfReplicas
	m.hashMap = make(map[int]string)
	if h == nil{
		m.hash = crc32.ChecksumIEEE
	}else {
		m.hash = h
	}
	return m
}

//AddNode 添加新的真实节点
func (c *Map) AddNode(NodeName ...string){
	for _,v := range NodeName{
		for i := 0;i < c.replicas;i++{
			number := int(c.hash([]byte(strconv.Itoa(i) + v)))
			c.keys = append(c.keys,number)
			c.hashMap[number] = v
		}
	}
	sort.Ints(c.keys)
}

//GetNode 根据key获得一个key-value值所需要的真实存储节点名称
func (c *Map) GetNode(key string) string{
	if c.keys == nil{
		return ""
	}
	num := int(c.hash([]byte(key)))

	//找到 >= 目标key值的第一个节点

	index := sort.Search(len(c.keys), func(i int) bool {
		return c.keys[i] >= num
	})
	index = index%len(c.keys)
	node := c.hashMap[c.keys[index]]
	return node
}



