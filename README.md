# Polaris Sidecar

## 介绍

polaris-sidecar 作为 polaris 的本地边车代理，提供两个可选功能模式：

- 本地 DNS：使用 DNS 解析的方式访问北极星上的服务
- 服务网格：通过劫持流量的方式实现服务发现和治理，开发侵入性低

用户可以选择其中一种模式进行接入polaris-sidecar，本文档介绍如何在虚拟机或者容器环境中安装和使用 polaris-sidecar。

## 本地DNS模式

### 提供功能

- 基于DNS的服务发现能力：直接通过域名```<service>.<namespace>.svc.polaris```进行拉取服务实例地址列表。
- 故障节点剔除能力：自动剔除不健康和隔离实例，保障业务可靠性。
- 标签路由能力：通过配置标签，通过标签筛选并返回满足标签规则的服务实例地址列表。

### 安装说明

#### 前提条件

- 在安装和使用 polaris-sidecar 之前，需要先安装北极星服务端，安装方式请参考[北极星服务端安装文档](https://polarismesh.cn/zh/doc/快速入门/安装服务端/安装单机版.html#单机版安装)。

#### 虚拟机环境下安装

1. 虚拟机安装过程需要使用root用户或者具有超级管理员权限的用户来执行，并且确保53（udp/tcp）端口没有被占用。
2. 需要从[Releases](https://github.com/polarismesh/polaris-sidecar/releases)下载最新版本的安装包。
3. 上传安装包到虚拟机环境中，并进行解压，进入解压后的目录。
```
unzip polaris-sidecar-release_$version.$os.$arch.zip
```
4. 修改polaris.yaml，写入北极星服务端的地址，端口号使用8091（GRPC端口）。
```
global:
  serverConnector:
    addresses:
      - 127.0.0.1:8091
```
5. 进入解压后的目录，执行tool/start.sh进行启动，然后执行tool/p.sh查看进程是否启动成功。
```
# bash tool/start.sh
# bash ./tool/p.sh
root     15318     1  0 Jan22 ?        00:07:50 ./polaris-sidecar start
```
6. 修改/etc/resolv.conf，在文件中添加```nameserver 127.0.0.1```，并且添加到所有的nameserver记录前面，如下：
```
; generated by /usr/sbin/dhclient-script
nameserver 127.0.0.1
nameserver x.x.x.x
```
7. 验证安装，使用格式为```<service>.<namespace>.svc.polaris```的域名进行访问，可以获得服务的IP地址。
```
# dig polaris.checker.polaris.svc.polaris

; <<>> DiG 9.9.4-RedHat-9.9.4-29.el7_2.2 <<>> polaris.checker.polaris.svc.polaris
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 10696
;; flags: qr aa rd; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;polaris.checker.polaris.svc.polaris. IN        A

;; ANSWER SECTION:
polaris.checker.polaris.svc.polaris. 10 IN AAAA ::ffff:9.134.15.118

;; Query time: 0 msec
;; SERVER: 127.0.0.1#53(127.0.0.1)
;; WHEN: Wed Jan 26 00:21:34 CST 2022
;; MSG SIZE  rcvd: 127
```

备注：如果需要使用域名的方式进行服务发现，则必须保证命名空间和服务名在北极星上都是以全小写字母进行注册，否则会寻址失败。

#### 容器环境下安装

支持中
