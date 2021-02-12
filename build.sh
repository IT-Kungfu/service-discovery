#!/usr/bin/env bash

go mod download
go mod vendor
cd vendor/github.com/coreos/etcd/
curl -Ls https://github.com/etcd-io/etcd/pull/11580.patch | patch -p1
curl -Ls https://github.com/corpix/etcd/commit/adb7dcade831f698e657bf1ccab6c138be05fb84.patch | patch -p1
