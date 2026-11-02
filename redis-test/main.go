package main

import (
	"net/http"
	"strconv"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	pool := goredis.NewPool(rdb)
	rs := redsync.New(pool)
	r := gin.Default()
	// 写一个测试接口
	r.GET("/get-name", func(c *gin.Context) {
		// Go的标准库需要context.Context 而gin提供的是*gin.Context
		//  正确的做法是c.Request.Context 获取底层请求的上下文
		ctx := c.Request.Context()
		userID := "1001"
		limitkey := "limit:1" + userID
		// 实现防刷功能 使用setnx锁 建立成功 返回true 建立失败 返回false
		status, err := rdb.SetNX(ctx, limitkey, "1",5*time.Second).Result()
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"msg": "系统错误"})
			return
		}
		if !status {
			c.JSON(http.StatusOK, gin.H{"msg": "5秒后尝试"})
			return 
		}
		rdb.Incr(ctx, "limit:1")
		c.JSON(200, gin.H{"msg": "点赞成功"})
	})
	r.POST("/buy", func(c *gin.Context) {
		ctx := c.Request.Context()
		// 给商品上锁 避免多个请求一起进来
		mutex := rs.NewMutex( "lock:product:1")
		if err := mutex.LockContext(ctx); err != nil {
			c.JSON(409, gin.H{"msg": "抢购失败，请稍后重试"})
			return
		}
		// 无论如何 一定要释放锁
		defer mutex.UnlockContext(ctx)
		stockStr, _ := rdb.Get(ctx, "stock:product:1").Result()
		stock, _  := strconv.Atoi(stockStr)
		if stock <= 0 {
			c.JSON(200, gin.H{"msg": "已售罄"})
		}
		rdb.Decr(ctx, "stock:product:1")
		c.JSON(200, gin.H{"msg": "抢购成功"})
	})
	r.Run(":8080")
}
