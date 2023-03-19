# 内存存储引擎设计

data 是最主要的对外暴露接口，除此之外，可能会暴露其他结构体，具体的还需设计完成之后再进行叙述

这里的时间表示都是 int64，原因是为了节省空间， time.Timer 占据了 24 个字节，然而当我们使用纳秒为单位表示时间，int64 仅占据了8个字节，每一个结构体就能节省16个字节。

data 的主要链路是:
```
└── data
    └── topic_map
        ├── topic
        │   ├── running
        │   ├── pending
        │   └── current
        └── topic
            ├── running
            ├── pending
            └── current
```

其中 topic_map，topic，running，pending，service都分别拥有自己的锁，这是为了细分锁的细粒度，避免大规模的程序拥塞