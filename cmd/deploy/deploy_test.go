package main

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	testingKube "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"testing"
)

func TestKubernetes_CreateDeploymentFail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the fake client.
	client := fake.NewSimpleClientset()

	// We will create an informer that writes added pods to a channel.
	pods := make(chan *v1.Pod, 1)
	podInformers := informers.NewSharedInformerFactory(client, 0)
	podInformer := podInformers.Core().V1().Pods().Informer()
	podInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*v1.Pod)
			t.Logf("pod added: %s/%s", pod.Namespace, pod.Name)
			pods <- pod
		},
	})

	// Make sure informers are running.
	podInformers.Start(ctx.Done())

	client.PrependReactor("create", "*", func(action testingKube.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, fmt.Errorf("fail to create kubernetes deployment")
	})

	err := createDeployment(client.AppsV1())
	if err == nil {
		t.Errorf("Expected error because we react with an error with create method")
		return
	}
}

func TestKubernetes_CreateDeploymentSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the fake client.
	client := fake.NewSimpleClientset()

	// We will create an informer that writes added pods to a channel.
	pods := make(chan *v1.Pod, 1)
	podInformers := informers.NewSharedInformerFactory(client, 0)
	podInformer := podInformers.Core().V1().Pods().Informer()
	podInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*v1.Pod)
			t.Logf("pod added: %s/%s", pod.Namespace, pod.Name)
			pods <- pod
		},
	})

	// Make sure informers are running.
	podInformers.Start(ctx.Done())

	err := createDeployment(client.AppsV1())
	if err != nil {
		t.Errorf("fail to create mocked deployment, %v", err)
		return
	}
}
