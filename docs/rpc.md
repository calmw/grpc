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
        