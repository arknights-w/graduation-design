# Graduation Design

这是我的毕业设计项目，内容是一个基于go语言开发的简易注册中心

## AIRFONE

airfone 传话筒

Dynamic Naming and Configuration Service [ 动态命名和配置服务 ]

这是一个服务注册/发现中心，采用 DDD 架构进行代码编写，目的是为了让服务中心中的组件具有高延展性，可插拔性。

(这里暂时不会涉及集群，因为毕业时间临近了，raft协议虽然原理不难，但是实现起来很麻烦，这里仅讨论一个注册中心应该拥有的一些功能，以及如何使用回调函数来自定义选择服务[一个服务可能有多个提供者，选择哪个最为合适]，达到一定的负载均衡效果)



todo: 
* 如何完成更高效的分配，比如根据网络信息，机房信息，地域信息进行个性化的服务依赖
* 如何处理上游服务失效，链式的导致大批量下游服务失效，并且需要多轮心跳进行恢复
* 更换 map，使用自己实现的更加高效的 map 类型
* 服务端使用可重入锁进行重构，代码逻辑链路更加清晰
* 客户端如何响应式的通知依赖变更(客户端能够接收到变更，但是)
* 客户端多语言平台适配
* 服务端多协议适配
* 服务端监控后台

## LogService

这是用来测试注册中心的额外服务

他的作用是对外提供日志服务，比如服务A希望他的日志能够被记录到远程的LogServie上，则调用该服务客户端获得服务

LogService 会被注册到注册中心中，依赖他的服务则是通过服务发现功能获取他的相关信息

## CommonService

这是一个简单的服务，他依赖于日志服务

他需要将自身注册到注册中心中，并获取日志服务依赖信息，再通过日志服务的客户端获取相关日志服务

## 跑项目的顺序

先运行服务发现中心，再运行日志服务，最终运行CommonService

todo: 或许可以通过 docker compose 进行部署
