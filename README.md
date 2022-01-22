# Polaris Sidecar

## QuickStart
本项目为北极星sidecar(边车)项目， 提供了 UDP 接口， 本地北极星名字服务agent相关功能。
注意：
* 运行需要root权限，会添加127.0.0.1到 本机的 /etc/resolv.conf文件（且必须添加到第一行），用于拦截域名解析
* 会在 crontab 中添加程序存活检查拉起定时任务


### 1.部署指南

#### 1. 下载可运行程序包
在本项目的git路径下 tag 栏目中下载运行程序包，软件包名字为：polaris-sidecar-${version}.tar.gz

需要直接进去对应的tag中，点击附件进行下载
[图片]![图片](/uploads/890D0EB974C44AC3BCCCF199F7357E5E/图片)

https://github.com/polarismesh/polaris-sidecar/-/tags

#### 2. 安装
把程序包上传到目标机器，解压后，进入tool目录
bash install.sh
会自动执行start, 无需执行start.sh

检查是否启动 OK：
* ps -ef | grep polaris-sidecar   
检查进程是否存在
* crontab -l  
检查 crontab 中是否有相关定时任务
* /etc/resolv.conf   
检查是否有 nameserver 127.0.0.1
#### 3. 停止
cd tool   
./stop.sh
* 停止后会去掉 /etc/resolv.conf   中 127.0.0.1行

#### 4. 启动
./start.sh

#### 5. 卸载
./uninstall.sh



### 2. 功能介绍
#### 1. dns 解析
对于在北极星官网注册的service, 提供标准的dns解析能力。对于使用第三方组件，可以在其配置文件中填写北极星service，从而提供名字服务能力。  
格式为   service_name.namespace.polaris  
如 在北极星官网中 注册了 Production   serviceA 服务， 那其北极星域名为  serviceA.Production.polaris   
注意：  
* 对于l5 的服务名，由于其不是有效DNS域名格式， 不能直接用L5使用该功能，可以创建一个该L5对应的北极星别名，从而可以使用该功能。

##### 1.常用工具支持情况
* ping  
支持
* host  
支持
* curl  
支持
* nslookup   
不完全支持，目前可以返回单个IP   
原因： 没有支持round-robin DNS
* dig  
不支持  
原因： 没有实现支持 EDNS


#### 2. 北极星标准功能
待补充




