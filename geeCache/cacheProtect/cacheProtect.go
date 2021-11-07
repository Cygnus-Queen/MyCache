package cacheProtect

import (
	"sync"
)

type cacheProtect struct {
	wg sync.WaitGroup
	value interface{}
	err error
}

type AccessRecord struct {
	record map[string]*cacheProtect
	accessMu sync.Mutex
}

func NewAccessRecord() *AccessRecord{
	a := new(AccessRecord)
	a.record = make(map[string]*cacheProtect)
	return a
}

func (a *AccessRecord) GetInstantaneousData(key string,fn func() (interface{},error)) (value interface{},err error){
	a.accessMu.Lock() //上锁，避免出现并发修改
	if ele,ok := a.record[key];ok{
		a.accessMu.Unlock() //属于瞬时数据，不会再进行修改，可以解锁
		ele.wg.Wait() //前面已经有瞬时数据正在远程请求，因此需要等待
		return ele.value,nil
	}

	//瞬时数据无记录，则新建记录
	c := new(cacheProtect)
	c.wg.Add(1) //上锁，正在远程请求数据，其他请求需要等待
	a.record[key] = c
	a.accessMu.Unlock() //a修改完成，解锁

	c.value,c.err = fn()
	c.wg.Done() //解锁，远程请求完毕

	a.accessMu.Lock() //修改a，上锁
	delete(a.record,key) //record中为瞬间到访的数据，不必长留
	a.accessMu.Unlock() //修改a结束，解锁

	return c.value,c.err
}

