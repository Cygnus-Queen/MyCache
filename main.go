package main

import (
	"MyRedis/geeCache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

//模拟数据库
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

//Get 模拟用户定义的真实的回调函数，从数据库中获取未命中的数据
func (temp *backGet) Get(key string) ([]byte, error) {
	log.Println("[SlowDB] search key", key)
	if v, ok := db[key]; ok {
		return []byte(v), nil
	}
	return nil, fmt.Errorf("%s not exist", key)
}

type backGet func(key string) ([]byte, error)

func startAPIServer(apiAddr string, g interface{},firstGetUrl string) {
	//私有结构体，使用空接口传参
	gee := geeCache.TransGroup(g)
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			log.Println("Request is handled by ", firstGetUrl)
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			//w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("front-end server is running at ", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func startCacheServer(addr string, adders []string, g interface{}) {
	gee := geeCache.TransGroup(g)
	peers := geeCache.NewHttpPool(addr)
	if adders != nil{
		peers.Set(adders...)
		gee.RegisterPeers(peers)
	}else {
		gee.RegisterPeers(nil)
	}

	log.Println("geecache is running at ", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func main(){
	//从命令行启动并指定cache的端口
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}
	firstGetUrl := addrMap[port]
	gee := geeCache.NewGroup("scores",2 << 10,new(backGet))
	if api {
		go startAPIServer(apiAddr, gee,firstGetUrl)
	}
	startCacheServer(firstGetUrl, []string(addrs), gee)

}