package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getBeforeTime() time.Time {
	minutes, err := strconv.Atoi(os.Getenv("DVORNIK_POD_AGE"))
	if err != nil {
		panic(err.Error())
	}
	if minutes <= 0 {
		panic("Pods age must be greater than 0")
	}
	return time.Now().Add(-time.Minute * time.Duration(minutes))
}

func getNamespace() string {
	namespace := os.Getenv("DVORNIK_NAMESPACE")
	if namespace == "" {
		panic("Please provide the namespace")
	}
	return namespace
}

func getClientset() kubernetes.Clientset {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("DVORNIK_KUBECONFIG"))
	if err != nil {
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
		return *clientset
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return *clientset
}

func getPods(clientset kubernetes.Clientset, namespace string) []corev1.Pod {
	opts := metav1.ListOptions{
		LabelSelector: os.Getenv("DVORNIK_LABEL_SELECTOR"),
	}
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), opts)
	if err != nil {
		panic(err.Error())
	}
	var filteredPods []corev1.Pod
	beforeTime := getBeforeTime()
	for _, pod := range pods.Items {
		if pod.ObjectMeta.CreationTimestamp.Time.Before(beforeTime) && (pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending) {
			filteredPods = append(filteredPods, pod)
		}
	}
	return filteredPods
}

func deletePods(pods []corev1.Pod, clientset kubernetes.Clientset, namespace string) {
	if len(pods) > 0 {
		fmt.Println("The following pods have been deleted:")
	}
	for _, pod := range pods {
		opts := metav1.DeleteOptions{}
		if pod.Status.Phase == corev1.PodPending {
			opts.GracePeriodSeconds = new(int64)
		}
		err := clientset.CoreV1().Pods(namespace).Delete(context.TODO(), pod.ObjectMeta.Name, opts)
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(pod.ObjectMeta.Name)
	}

}

func main() {
	namespace := getNamespace()
	clientset := getClientset()
	pods := getPods(clientset, namespace)
	deletePods(pods, clientset, namespace)
}
