//package main
//
//import (
//	"context"
//	"github.com/coreos/etcd/mvcc/mvccpb"
//	"go.etcd.io/etcd/clientv3"
//	"log"
//	"os"
//	"os/signal"
//	"sync"
//	"syscall"
//	"time"
//)
//
//type ServiceDiscovery struct {
//	cli        *clientv3.Client
//	serverList map[string]string
//	lock       sync.RWMutex
//}
//
//func NewServiceDiscovery(endpoints []string) *ServiceDiscovery {
//	cli, err := clientv3.New(clientv3.Config{
//		Endpoints:   endpoints,
//		DialTimeout: 5 * time.Second,
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	return &ServiceDiscovery{
//		cli:        cli,
//		serverList: make(map[string]string),
//	}
//}
//func (s *ServiceDiscovery) WatchService(prefix string) error {
//	resp, err := s.cli.Get(context.Background(), prefix, clientv3.WithPrefix())
//	if err != nil {
//		return err
//	}
//	for _, ev := range resp.Kvs {
//		s.SetServiceList(string(ev.Key), string(ev.Value))
//	}
//	go s.watcher(prefix)
//	return nil
//}
//
//func (s *ServiceDiscovery) SetServiceList(key string, val string) {
//	s.lock.Lock()
//	defer s.lock.Unlock()
//	s.serverList[key] = string(val)
//	log.Println("put key :", key, "val:", val)
//}
//
//func (s *ServiceDiscovery) watcher(prefix string) {
//	rch := s.cli.Watch(context.Background(), prefix, clientv3.WithPrefix())
//	log.Printf("watching prefix:%s now...", prefix)
//	for wresp := range rch {
//		for _, ev := range wresp.Events {
//			switch ev.Type {
//			case mvccpb.PUT:
//				s.SetServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
//			case mvccpb.DELETE:
//				s.DelServiceList(string(ev.Kv.Key))
//			}
//		}
//	}
//}
//
//func (s *ServiceDiscovery) DelServiceList(key string) {
//	s.lock.Lock()
//	defer s.lock.Unlock()
//	delete(s.serverList, key)
//	log.Println("del key:", key)
//}
//func (s *ServiceDiscovery) GetServices() []string {
//	s.lock.RLock()
//	defer s.lock.RUnlock()
//	addrs:=make([]string,0,len(s.serverList))
//	for _,v:=range s.serverList{
//		addrs=append(addrs,v)
//	}
//	return addrs
//}
//func (s *ServiceDiscovery) Close() error {
//	return s.cli.Close()
//}
//func main() {
//	var endpoints=[]string{"localhost:2379"}
//	ser:=NewServiceDiscovery(endpoints)
//	defer func() {
//		if err:=ser.Close();err!=nil{
//			log.Fatalln(err)
//		}
//	}()
//	err:=ser.WatchService("/server/")
//	if err!=nil{
//		log.Fatal(err)
//	}
//	c:=make(chan os.Signal,1)
//	go func() {
//		signal.Notify(c,syscall.SIGINT,syscall.SIGTERM)
//	}()
//	for{
//		select {
//		case <-time.Tick(10 * time.Second):
//			log.Fatalln(ser.GetServices())
//		case <-c:
//			log.Fatalln("server discovery exit")
//			return
//		}
//	}
//}

package main

import (
	"os"
	"log"
	"time"
	"sync"
	"syscall"
	"context"
	"os/signal"

	"go.etcd.io/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
)

//ServiceDiscovery ????????????
type ServiceDiscovery struct {
	cli        *clientv3.Client  //etcd client
	serverList map[string]string //????????????
	lock       sync.RWMutex
}

//NewServiceDiscovery  ??????????????????
func NewServiceDiscovery(endpoints []string) *ServiceDiscovery {
	//?????????etcd client v3
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		log.Fatal(err)
	}

	return &ServiceDiscovery{
		cli:        cli,
		serverList: make(map[string]string),
	}
}

//WatchService ??????????????????????????????
func (s *ServiceDiscovery) WatchService(prefix string) error {
	//???????????????????????????key
	resp, err := s.cli.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	//??????????????????key???value
	for _, ev := range resp.Kvs {
		s.SetServiceList(string(ev.Key), string(ev.Value))
	}

	//??????????????????????????????server
	go s.watcher(prefix)
	return nil
}

//watcher ??????key?????????
func (s *ServiceDiscovery) watcher(prefix string) {
	rch := s.cli.Watch(context.Background(), prefix, clientv3.WithPrefix())
	log.Printf("watching prefix:%s now...", prefix)
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT: //??????????????????
				s.SetServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE: //??????
				s.DelServiceList(string(ev.Kv.Key))
			}
		}
	}
}

//SetServiceList ??????????????????
func (s *ServiceDiscovery) SetServiceList(key, val string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.serverList[key] = string(val)
	log.Println("put key :", key, "val:", val)
}

//DelServiceList ??????????????????
func (s *ServiceDiscovery) DelServiceList(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.serverList, key)
	log.Println("del key:", key)
}

//GetServices ??????????????????
func (s *ServiceDiscovery) GetServices() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	addrs := make([]string, 0,len(s.serverList))

	for _, v := range s.serverList {
		addrs = append(addrs, v)
	}
	return addrs
}

//Close ????????????
func (s *ServiceDiscovery) Close() error {
	return s.cli.Close()
}

func main() {
	var endpoints = []string{"localhost:2379"}
	ser := NewServiceDiscovery(endpoints)
	defer ser.Close()

	err := ser.WatchService("/server/")
	if err != nil {
		log.Fatal(err)
	}


	// ??????????????????????????? ctrl + c ??????????????????????????????
	c := make(chan os.Signal, 1)
	go func() {
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	}()

	for {
		select {
		case <-time.Tick(10 * time.Second):
			log.Println(ser.GetServices())
		case <-c:
			log.Println("server discovery exit")
			return
		}
	}
}