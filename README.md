[English](./README.md) | [简体中文](./README.zh-CN.md)
gomule
======

Goroutines eMule Server

install 
=======
```
go install github.com/test3-damianfurrer/gomule@latest
```
binary will be in the ```go env GOBIN``` path

sql db used is according to eNode:
https://github.com/zt8989/eNode/blob/5fb46f1e2a64ce91c274f6cbb85e854f6aa3f6dc/misc/enode_2019-05-26.sql

usage dev
=====
* go run gomule.go -d 
* go run gomule.go -h 10.0.0.159 -p 7771

screenshot 
==========
![ScreenShot](https://raw.github.com/xiangzhai/gomule/master/doc/login.png)
