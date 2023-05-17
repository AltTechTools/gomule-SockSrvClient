gomule
======

Goroutines eMule Server

安装  
=======
1.创建数据库

https://github.com/zt8989/eNode/blob/5fb46f1e2a64ce91c274f6cbb85e854f6aa3f6dc/misc/enode_2019-05-26.sql

2.下载源码编译
```
go build
```

3.运行
```
./gomule -d -h 0.0.0.0 -p 7771 -us -su root -pw 123456 -db gomule
```
注意这里需要有一个MySQL数据库，用户名root，密码123456，有一个gomule的库，ip 127.0.0.1，端口3306


参数
=======

| 字段 | 类型     | 说明                |
|----|--------|-------------------|
| d  | bool   | 是否输出调试信息          |
| h  | ip     | 监听地址，默认 localhost |
| p  | 端口     | 服务器端口，默认7111      |
| i  | bool   | 使用i2p             |
| s  | ip     | SAM 的地址           |
| v  | bool   | 输出版本号             |
| us | bool   | 使用sql数据库          |
| sd | string | 使用数据库渠道，默认mysql   |
| su | string | 数据库用户名，默认user     |
| pw | string   | 数据库密码，默认password  |
| db | string   | 数据库名，默认gomule     |
| ssi | ip   | 数据库ip或者域名         |
| ssp | 端口   | 数据库d端口            |




usage dev
=====
* go run gomule.go -d 
* go run gomule.go -h 10.0.0.159 -p 7771

screenshot 
==========
![ScreenShot](https://raw.github.com/xiangzhai/gomule/master/doc/login.png)
