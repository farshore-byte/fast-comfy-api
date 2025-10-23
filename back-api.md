# 核心api

prompt json + comfyui-nodes

进入api 

并行执行以下操作：

- 检查comfyui-nodes中的服务器是否都已经建立了wscoket连接，如果没有，则立即建立连接
- 并发调用每个服务器的/queue接口，查看当前队列长度，选择一个具有最短队列的服务器，并将请求发送到该服务器，并获取到prompt_id
- websocket连接建立后，comfyui服务器会将prompt_id提示完成的消息推送到go，go监听接收到某个prompt_id完成后，立即调用回调api, /recall/:prompt_id
- 回调接口查询生成结果，上传s3, 组织调用结果，返回

Go 的 channel（通道） 是用来在不同 goroutine（协程）之间安全传递数据的机制。


它就像一个 有线管道，一个 goroutine 可以往里面放数据，另一个 goroutine 从另一端取出来。 


接口中的请求使用 channel实现阻塞，从channel获取到全部结果才会结束
