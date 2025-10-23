# api 
## 存储形式 json

## 接口路由统一为  当前go运行ip 地址 + 端口号 + /v1/generate   

例如 go 运行在 192.168.1.100 端口号为 8080 的情况下，接口路由为：http://192.168.1.100:8080/v1/generate

每个api都分配一个token, 接口请求时需要携带token，由此路由到对应的生成任务api

## 要素组成：
- 接口名
- 接口描述
- prompt json 
- variables 替换变量路径
- comfyui nodes 一个服务地址的列表



# 前端 （mvp）
三步走：
1. 上传prompt json 文件
2. 配置一个或者多个comfyui服务器地址
3. 将json中的某些字段选择为变量，左侧勾选字段，右侧动态生成API文档，可在线调试
4. 发布api，并生成api json文档


# 后端

## 前端配置时
前端到配置服务器完成后，点击下一步时，立即和服务器建立websocket连接，连接成功则进入下一步，否则提示错误
到第三步，前端将以下参数组织成request 请求后端
- 拼装好的最终的prompt
- comfyui_nodes 列表

后端提供一个api，接收这些参数，生成结果，上传s3, 返回结果连接固定的响应格式


##  API 服务

统一路由 /v1/generate 

识别token, 读取对应的api json 文件

根据变量路径，对prompt json中的变量进行替换

调用 后端提供的api, 接收prompt json, comfyui_nodes 列表, 生成结果，上传s3, 返回结果连接固定的响应格式



