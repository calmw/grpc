#### go中的rpc包

    使用net/rpc包的一个直接限制是客户端和服务端必须都用go编写。默认情况下，数据交换使用go特定的gob格式进行
    作为对net/rpc的改进，net/rpc/jsonrpc包允许我们使用JSON作为HTTP上的数据交换格式。因此服务器现在可以用Go编写，但客户端不用。
    与标准库的rpc包相比，通用RPC框架的主要优势在于它使我们能够使用不同的编程语言写服务端和客户端应用程序

#### grpc的使用

    1)使用grpc创建应用程序的第一步是使用协议缓冲区语言定义服务接口
    2)生成go文件
        protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative users.proto
        --go_out和go-grpc_out选项指定文件的生成路径，这里指定为当前目录。--go_opt=paths=source_relative 和--go-grpc_opt=paths=source_relative 指定文件应该根据users.proto文件的位置生成。
        我们永远不要手动编辑这些文件

#### 安装编译工具

    Mac:
        brew install protobuf
        brew install protoc-gen-go-grpc

#### gRPC 服务端健康检查

    当服务器启动时，可能需要几秒钟的时间来创建网络侦听器、注册gRPC服务并建立与数据存储或其他服务的连接。因此，它可能不会立即准备好处理客户端请求。
    最重要的是，服务器在运行期间可能会因为请求而变得过载，以至于它不应该真正接受任何新的请求。
    在以上两种情况下，建议在服务中添加一个RPC方法，用于探测服务器是否健康。通常此探测将由另一个应用程序执行，例如负载均衡或代理服务，它们根据运行状况探测是否成功将请求转发到服务器。
    gRPC健康检查协议定义了专用的服务规范，参考health.proto文件
        