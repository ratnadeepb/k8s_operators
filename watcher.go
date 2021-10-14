// package main

// import (
// 	"context"
// 	"flag"

// 	apiv1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/client-go/kubernetes"
// 	"k8s.io/client-go/tools/clientcmd"
// )

// func errHandle(err error) {
// 	if err != nil {
// 		panic(err.Error())
// 	}
// }

// func main() {
	// config, err := clientcmd.BuildConfigFromFlags("", "/users/deep83/.kube/config")
	// errHandle(err)

	// clientset, err := kubernetes.NewForConfig(config)
	// errHandle(err)

	// nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	// errHandle(err)

	// println("Nodes")
	// for _, node := range nodes.Items {
	// 	println(node.Name)
	// }

	// eps, err := clientset.CoreV1().Endpoints("default").List(context.Background(), metav1.ListOptions{})
	// errHandle(err)
	// println("Endpoints")
	// for _, ep := range eps.Items {
	// 	println(ep.Name)
	// }

	// deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

	// println("Deployments")
	// deps, err := deploymentsClient.List(context.Background(), metav1.ListOptions{})
	// errHandle(err)
	// for _, dep := range deps.Items {
	// 	println("%s %d", dep.Name, *dep.Spec.Replicas)
	// }

	// pods, err := clientset.CoreV1().Pods("default").List(context.Background(), metav1.ListOptions{LabelSelector: "app=testapp-svc-0"})
	// errHandle(err)
	// println("Pods")
	// for _, pod := range pods.Items {
	// 	println(pod.Name)
	// 	for key, val := range pod.Labels {
	// 		println(key, val)
	// 	}
	// }
// }
