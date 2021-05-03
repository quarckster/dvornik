package main

import (
	"context"
	"encoding/json"
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

func getExceptionLabels() map[string]string {
	labels := []byte(os.Getenv("DVORNIK_EXCEPTIONS"))
	var exceptionLabels map[string]string
	if len(labels) != 0 {
		err := json.Unmarshal(labels, &exceptionLabels)
		if err != nil {
			panic(err.Error())
		}
	}
	return exceptionLabels
}

func checkPod(pod corev1.Pod, exceptionLabels map[string]string, beforeTime time.Time) bool {
	if pod.ObjectMeta.CreationTimestamp.Time.Before(beforeTime) && pod.Status.Phase == corev1.PodRunning {
		shouldSkipped := false
		for podLabelKey, podLabelValue := range pod.ObjectMeta.Labels {
			if exceptionLabelValue, ok := exceptionLabels[podLabelKey]; ok {
				shouldSkipped = exceptionLabelValue == podLabelValue
				break
			}
		}
		return !shouldSkipped
	}
	return false
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
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	var filteredPods []corev1.Pod
	exceptionLabels := getExceptionLabels()
	beforeTime := getBeforeTime()
	for _, pod := range pods.Items {
		if checkPod(pod, exceptionLabels, beforeTime) {
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
		err := clientset.CoreV1().Pods(namespace).Delete(context.TODO(), pod.ObjectMeta.Name, metav1.DeleteOptions{})
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
