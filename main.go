package main

import (
	"deploy/filetype"
	"errors"
	"flag"
	"github.com/golang/glog"
	"io/ioutil"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strings"
	"time"
)

const (
	kubeconfigName = "kubeconfig"
)

func main() {
	// 在指定位置找到kube-config文件
	// 在指定位置找到yaml文件
	// 执行apply
	var (
		kubeconfigPath string
		yamlPath string
	)
	// 获取参数变量
	flag.StringVar(&kubeconfigPath, "kubeconfig_path", "./test-project/deploy/develop/kubeconfig", "kubeconfig_path")
	flag.StringVar(&yamlPath, "yaml_path", "./test-project/deploy/develop/yaml", "yaml_path")
	flag.Parse()
	// 检测指定路径是否存在文件
	if err := filetype.HasFileInConfPath(yamlPath, ""); err != nil {
		glog.Errorf("HasYamlInConfPath failed, error is %s", err.Error())
		return
	}
	if err := filetype.HasFileInConfPath(kubeconfigPath, kubeconfigName); err != nil {
		glog.Errorf("HasYamlInConfPath failed, error is %s", err.Error())
		return
	}
	if err := deployToK8s(kubeconfigPath, yamlPath); err != nil {
		glog.Errorf("deploy failed, error is %s", err)
		return
	}
	//
	glog.Info("deploy is finished! ")
}

func deployToK8s(kubeconfigPath, yamlFilePath string ) error {
	if len(kubeconfigPath) == 0 || len(yamlFilePath) == 0 {
		return errors.New("kubeconfigPath or yamlFilePath are empty! ")
	}
	// 下面开始使用client-go去部署
	kubeconfig , err := clientcmd.BuildConfigFromFlags("", kubeconfigPath + "/kubeconfig")
	if err != nil {
		glog.Errorf("BuildConfigFromFlags error, error is %s\n", err.Error())
		return err
	}
	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		glog.Errorf("NewForConfig error, error is %s\n", err.Error())
		return err
	}
	// 读取
	dirList, err := ioutil.ReadDir(yamlFilePath)
	if err != nil { //有错误，后面就不继续了
		glog.Errorf("readd dir failed, error is %s, dir is: %s", err.Error(), yamlFilePath)
		return err
	}
	return deploy(clientset, dirList, kubeconfig, yamlFilePath)
}

func deploy(client *kubernetes.Clientset, files []os.FileInfo, kubeconfig *rest.Config, yamlFilePath string) error {
	if client == nil || kubeconfig == nil {
		return errors.New("client, or kubeconfig is nil")
	}
	for _, v := range files {
		if strings.Contains(v.Name(), "yaml") || strings.Contains(v.Name(), "yml") {// 只对yaml文件进行操作
			f, err := os.Open(yamlFilePath +"/" + v.Name())
			if err != nil {
				return err
			}
			d := yaml.NewYAMLOrJSONDecoder(f, 4096)
			dc := client.Discovery()
			restMapperRes, err := restmapper.GetAPIGroupResources(dc)
			if err != nil {
				glog.Errorf("GetAPIGroupResources error, error is %s\n", err.Error())
				return err
			}
			restMapper := restmapper.NewDiscoveryRESTMapper(restMapperRes)
			namespace := "default"  // 默认default
			ext := runtime.RawExtension{}
			if err := d.Decode(&ext); err != nil {
				glog.Errorf("Decode error, error is %s\n", err.Error())
				return err
			}
			// runtime.Object
			obj, gvk, err := unstructured.UnstructuredJSONScheme.Decode(ext.Raw, nil, nil)

			mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				glog.Errorf("RESTMapping error, error is %s\n", err.Error())
				return err
			}

			// runtime.Object转换为unstructed
			unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
			if err != nil {
				glog.Errorf("DefaultUnstructuredConverter error, error is %s\n", err.Error())
				return err
			}

			var unstruct unstructured.Unstructured
			unstruct.Object = unstructuredObj
			if md, ok := unstruct.Object["metadata"]; ok {
				metadata := md.(map[string]interface{})
				if internalns, ok := metadata["namespace"]; ok {
					namespace = internalns.(string)
				}
			}

			// 动态客户端创建
			dclient, err := dynamic.NewForConfig(kubeconfig)
			if err != nil {
				glog.Errorf("dynamic.NewForConfig error, error is %s\n", err.Error())
				return err
			}

			// 需要做下逻辑判断，先查询该资源是否存在，存在就更新，更新若是返回无更新内容，就先删除，在创建

			_, err = dclient.Resource(mapping.Resource).Namespace(namespace).Create(&unstruct, metav1.CreateOptions{})
			if err != nil {
				glog.Errorf("create  resource error, error is %s\n", err.Error())
				if k8sErrors.IsAlreadyExists(err) || k8sErrors.IsInvalid(err) {// 已经存在那就删除重新创建 或者port被占用
					if err := dclient.Resource(mapping.Resource).Namespace(namespace).Delete(unstruct.GetName(), &metav1.DeleteOptions{}); err != nil {
						glog.Errorf("delete  resource error, error is %s\n", err.Error())
						return err
					}
					//删除操作发生后，资源不会立即清理掉，此时创建会存在is being deleting的错误
					if err = pollGetResource(mapping, dclient, unstruct, namespace); err != nil {
						glog.Errorf("yaml file is %s, delete  resource error, error is %s\n", v, err.Error())
						return err
					}
					_, err = dclient.Resource(mapping.Resource).Namespace(namespace).Create(&unstruct, metav1.CreateOptions{})
					if err != nil {
						glog.Errorf("create  resource error, error is %s\n", err.Error())
						return err
					}
				} else {
					return err
				}
			}
		}
	}
	return nil
}

func pollGetResource(mapping *meta.RESTMapping, dclient dynamic.Interface, unstruct unstructured.Unstructured, namespace string) error {
	for {
		select {
		case <-time.After(time.Minute * 10):
			glog.Error("pollGetResource time out, something wrong with k8s resource")
			return errors.New("ErrWaitTimeout")
		default:
			resource, err := dclient.Resource(mapping.Resource).Namespace(namespace).Get(unstruct.GetName(), metav1.GetOptions{})
			if err != nil {
				if k8sErrors.IsNotFound(err) {
					glog.Warningf("kind is %s, namespace is %s, name is %s, this app has been deleted.", unstruct.GetKind(), namespace, unstruct.GetName())
					return  nil
				} else {
					glog.Errorf("get resource failed, error is %s", err.Error())
					return err
				}
			}
			if resource != nil && resource.GetName() == unstruct.GetName() {// 判断资源存在，或者获取资源报错。
				glog.Errorf("kind: %s, namespace: %s, name %s still  exists", unstruct.GetKind(), namespace, unstruct.GetName())
			}
		}
		time.Sleep(time.Second * 2)// 2秒循环一次
	}
}
