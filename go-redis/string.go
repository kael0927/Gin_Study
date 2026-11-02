package main

import (
	_"context"
	"fmt"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 1. 连接本地Redis
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	//ctx := context.Background()
	// 适配器包装
	pool := goredis.NewPool(rdb)
	// 创建redsync实例
	rs := redsync.New(pool)
	// 创建一把锁
	mutex := rs.NewMutex("my-global-lock")

	err := mutex.Lock()
	if err != nil {
		fmt.Println("没抢到锁")
		return
	}
	fmt.Println("抢到锁")
	// 执行业务
	mutex.Unlock()
	fmt.Println("释放锁")
	
	
}
