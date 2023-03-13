# grpc

    健康检查demo
    用于探测服务器是否健康
    Check方法，客户端在必要时候可以定时调用Check来检查服务端健康变化
    Watch， 当服务端更改状态时（例如调用updateServiceHealth更改），客户端就会收到状态变化