# 使用client-go部署yaml文件
## 使用方式
工程内的test-project是我的示例，你可以直接指定yaml和kubeconfig路径即可
克隆仓库&构建镜像
```go
git clone https://github.com/fishingfly/deployYamlByClient-go.git
cd deployYamlByClient-go
export GO111MODULE=on 
export GOPROXY=https://goproxy.io
go mod vendor
make build
make image
```
如何将该镜像应用在tekton流水线中，请关注微信公众号”云原生手记“查看github CICD篇。公众号内有博主联系方式，欢迎骚扰！
