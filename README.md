# MyCNI

1.IPAM分配网络
2.实现同一个节点上Pod之间通信
3.实现跨节点Pod之间通信

![Pod.jpg](./assets/1656680087400-%E8%B7%A8%E8%8A%82%E7%82%B9Pod%E7%BD%91%E7%BB%9C.jpg)

参考实现：
https://www.dandelioncloud.cn/article/details/1505089235286847490

以上就是host-gw模式的通信原理。其核心就是将每个 Flannel 子网（Flannel Subnet，比 如：10.244.1.0/24）的“下一跳”，设置成了该子网对应的宿主机的 IP 地址。也就是说，这台“主机”（Host）会充当这条容器通信路径里的“网关”（Gateway）。这也 正是“host-gw”的含义。
当然，Flannel 子网和主机的信息，都是保存在 Etcd 当中的。flanneld 只需要 WACTH 这些数 据的变化，然后实时更新路由表即可。
host-gw 模式能够正常工作的核心，就在于 IP 包在封 装成帧发送出去的时候，会使用路由表里的“下一跳”来设置目的 MAC 地址。这样，它就会经 过二层网络到达目的宿主机。这就要求集群宿主机之间必须要二层连通的。
