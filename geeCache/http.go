package geeCache

import (
	"MyRedis/geeCache/consistenthash"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
)

//默认路由
const (
	defaultBasePath = "/_geeCache/"
	defaultReplicas = 3
)

//HttpPool 建立一个服务端，方便分布式缓存
type HttpPool struct {
	self string                     //自身的url
	basePath string                 //url除开GroupName与key的公共前缀
	httpMu sync.Mutex  				    //修改Hash环时需要上锁，hash环的实现文件只负责实现逻辑，不负责上锁
	peers *consistenthash.Map        //维护一个一致性Hash的指针
	peerMap map[string]*httpGetter  //url字符串 -> http客户端
}

//Set 更新hash环，并建立hash环中真实节点名称到其他服务端的映射
func (h *HttpPool) Set(nodes ...string){
	h.httpMu.Lock()
	defer h.httpMu.Unlock()
	h.peers = consistenthash.NewMap(defaultReplicas,nil)
	h.peers.AddNode(nodes...)
	h.peerMap = make(map[string]*httpGetter)
	for _,v := range nodes{
		h.peerMap[v] = &httpGetter{baseUrl: v + h.basePath}
		fmt.Println(v + h.basePath)
	}
}

func (h *HttpPool) pickPeer(key string) (DataGetter ,bool){
	h.httpMu.Lock()
	defer h.httpMu.Unlock()
	node := h.peers.GetNode(key)
	log.Println("[cache miss] Then get to " + node,"     " + h.self)
	//先在自己的cahce下查找，找不到就去其他cache，因此若node与本cache地址一致，则失效，需要从自己的本地查找
	if node != "" && node != h.self{
		if p,ok := h.peerMap[node];ok{
			return p,true
		}
		log.Println("[httpGetter can not get] from " + node)
	}
	return nil,false
}

func NewHttpPool(s string) *HttpPool{
	httpPool := new(HttpPool)
	httpPool.basePath = defaultBasePath
	httpPool.self = s
	return httpPool
}

func (h *HttpPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Server %s] %s",r.Method,h.self + r.URL.Path)

	// /<basepath>/<groupname>/<key> required
	//为了控制url使用SplitN来进行url切分

	parts := strings.SplitN(r.URL.Path[len(h.basePath):],"/",2)
	if len(parts) != 2{
		http.Error(w, "The get method request of url is wrong", http.StatusNotFound)
	}

	groupName := parts[0]
	key := parts[1]

	if group ,ok := groups[groupName];ok{
		v ,err := group.Get(key)

		if err != nil{
			http.Error(w,err.Error(),http.StatusInternalServerError)
		}
		_, err = w.Write(v.ByteSlice())
		if err != nil {
			http.Error(w,err.Error(),http.StatusInternalServerError)
		}else {
			return
		}
	}else {
		http.Error(w, "No such group: "+groupName, http.StatusNotFound)
	}
}

//httpGetter 客户端，实现DataGetter接口
type httpGetter struct {
	baseUrl string //表示将要访问的远程节点的url
}

//Get 实现peerGetter接口
func (h *httpGetter) Get(GroupName string, key string) ([]byte, error) {
	u := fmt.Sprintf(
			"%s%s/%s",
			h.baseUrl,
			GroupName,
			key,
		)
	res,err := http.Get(u)
	//Get检验
	if err != nil{
		return nil,err
	}

	//相应内容检验
	if res.StatusCode != http.StatusOK{
		return nil,fmt.Errorf(" | server return wrong, %v",res.Status)
	}

	//将reader转换为[]byte
	ans ,err := ioutil.ReadAll(res.Body)
	if err != nil{
		return nil,fmt.Errorf("| ReadALL error, %v",err)
	}
	return ans,nil
}




