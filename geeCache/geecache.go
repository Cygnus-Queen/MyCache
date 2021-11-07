package geeCache

import (
	"MyRedis/geeCache/cacheProtect"
	"fmt"
	"log"
	"sync"
)

var (
	mu sync.RWMutex
	groups = make(map[string]*group)    //group池，根据实际场景可以建立多个group，并使用string进行索引
)

func GetGroup(key string) *group{
	mu.RLocker()
	defer mu.RUnlock()
	if ele,ok := groups[key];ok{
		return ele
	}
	return nil
}

func SetGroup(key string,g *group){
	mu.Lock()
	defer mu.Unlock()
	groups[key] = g
}


type Getter interface {
	Get(string) ([]byte,error)
}

type GetterFunc func(string) ([]byte,error)

func (c GetterFunc) Get(key string) ([]byte,error){
	return c(key)
}

//group 继续封装cache,添加编号和回调函数
type group struct {
	mainCache cache       //cache
	getter    Getter      //缓存未命中时的回调函数
	Name      string      //cache的id
	peers	  peerPicker  //我httpPool，包括了一个hash环
	record    *cacheProtect.AccessRecord //记录瞬时数据，瞬间返回，防止缓存穿透和击穿
}

func NewGroup(name string,MaxBytes int64,getter Getter) *group{
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()

	//创建新的group
	temp := new(group)
	temp.mainCache = cache{
		cacheBytes: MaxBytes,
	}
	temp.getter = getter
	temp.Name = name
	temp.record = cacheProtect.NewAccessRecord()
	//将group添加到groups中
	groups[name] = temp


	return temp
}

//TransGroup group为私有结构体，方便其他包进行空接口传参
func TransGroup(g interface{}) *group{
	gee := g.(*group)
	return gee
}

//RegisterPeers 注册group.peers
func (c *group) RegisterPeers(peers peerPicker){
	if peers == nil{
		c.peers = nil
	}
	c.peers = peers
}

func (c *group) Get(key string) (ByteView, error){
	if key == "" {
		return ByteView{}, fmt.Errorf("key is empty")
	}

	if ele,ok := c.mainCache.get(key);ok{
		log.Println("[ GeeCache hit ]" + "| value is " + string(ele.ByteSlice())) //成功查找到对应的值
		return ele,nil
	}else {
		return c.Load(key) //分为从其他集群获取和本地获取，因此需要降低耦合
	}
}

//Load 根据应用场景的不同，需要分类为从本地获取和从其他分布式集群的cache获取
func (c *group) Load(key string) (ByteView,error){
	ele ,err := c.record.GetInstantaneousData(key, func() (interface{}, error) {
		if c.peers != nil{
			if peer,ok := c.peers.pickPeer(key);ok{
				ans, err := peer.Get(c.Name,key)
				if err == nil {
					x := ByteView{cloneByte(ans)}
					return x,nil
					//return ByteView{}, err
				}
				log.Println("[GeeCache] Failed to get from data", err)
			}
		}
		return c.getLocally(key)
	})
	return ele.(ByteView),err

}

//getLocally cache未命中，从本地获取数据并更新cache
func (c *group) getLocally(key string) (ByteView, error){
	ele ,err := c.getter.Get(key)
	if err != nil{
		log.Println("Can not find key in DB")
		return ByteView{},err
	}
	//ele是一个字符串，从其他地方获取的数据无法保证适用于本cache，因此需要类型转换
	v := ByteView{b: cloneByte(ele)} //防止更改底层数据，因此要进行数据拷贝
	c.mainCache.add(key,v)
	return v,nil
}

type DataGetter interface {
	Get(name string,key string) ([]byte, error)
}

type peerPicker interface {
	pickPeer(key string) (p DataGetter ,ok bool)
}




