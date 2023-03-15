# grpc

    服务端Panic和超时处理
    panic处理拦截器一定要放在最内侧（写在最后，要不然服务端还是有可能会Panic）。将panic处理拦截器注册为最内层拦截器，意味着服务处理程序中的运行时错误不会妨碍其他拦截器的功能
    一元RPC直接写拦截器来处理超时，流RPC超时处理需要包装一下，代码如下：

```
type wrappedServerStream struct {
RecvMsgTimeout time.Duration // 增加超时，并修改RecvMsg方法
grpc.ServerStream
}
```

    在流RPC方法的情况下，流连接很可能是长期存在的。请求和响应应包含一个消息流，在流上的连接消息之间可能存在延迟。对流RPC方法进行强制超时该怎么办？当服务器等待接收消息并且计时器将根据消息重置时，将实施最大超时策略。包装来自底层的ServerStream对象，并在RecvMsg中实现逻辑
    测试panic处理拦截器修改：

```go
result, err := c.GetUser(context.Background(), &svc.UserGetRequest{
//Email: "panic@doe.com", // 测试服务端panic @前为panic时触发服务端panic，其它正常
Email: "cisco@doe.com", // @前为panic时触发服务端panic，其它正常
})
```

    优雅关机：
        可以在接收到系统信号或者根据情况调用服务端编写的stopServer函数优雅关机（关闭服务端）
        没有采用GracefulStop是因为，这个方法不允许调用者配置最大超时时间。所以采用 1 更新健康检查状态为NOT_SERVING 2 等待一定时间，尝试将现有请求处理完，3 调用Stop方法进行硬关机