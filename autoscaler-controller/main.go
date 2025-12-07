package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/common/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const (
	MIN_REPLICAS = 1
	MAX_REPLICAS = 5
	// Target average tasks per replica (TTR)
	TARGET_TTR     = 10.0
	SCALE_INTERVAL = 15 * time.Second
)

const prometheusURL = "http://prometheus-service:9090"

func calculateNextReplicas(currentReplicas int32, activeTasks float64) int32 {
	desiredReplicasFloat := activeTasks / TARGET_TTR
	desiredReplicas := int32(math.Ceil(desiredReplicasFloat))

	if desiredReplicas > MAX_REPLICAS {
		return MAX_REPLICAS
	} else if desiredReplicas < MIN_REPLICAS {
		return MIN_REPLICAS
	}

	return desiredReplicas
}

func setupPrometheusClient() (v1.API, error) {
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, err
	}
	log.Printf("Client Connected")

	promAPI := v1.NewAPI(client)

	return promAPI, nil
}

func setupK8sClient() (*kubernetes.Clientset, error) {

	config, err := rest.InClusterConfig()

	if err != nil {
		home := homedir.HomeDir()
		kubeconfigPath := filepath.Join(home, ".kube", "config")

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create k8s config: %w", err)
		}
	}

	if err != nil {
		return nil, err
	}
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	log.Println("Client Created")

	return k8sClient, nil
}

func main() {

	promAPI, err := setupPrometheusClient()
	if err != nil {
		log.Fatalf("Couldnt Create Prometheus Client", err)
	}

	k8sClient, err := setupK8sClient()
	if err != nil {
		log.Fatalf("Couldnt Create Kubernetes Client", err)
	}

	for {
		log.Println("---Starting Scaling Iteration---")

		query := "sum(watcherBot_active_tasks)"
		result, warnings, err := promAPI.Query(context.Background(), query, time.Now())

		if err != nil {
			log.Fatalf("Error querying Prometheus %v\n", err)
			time.Sleep(SCALE_INTERVAL)
			continue
		}

		if len(warnings) > 0 {
			log.Printf("Prometheus query returned warnings %v\n", warnings)
		}

		fmt.Printf("Query successful. Result Type: %T\n", result)

		vector, ok := result.(model.Vector)
		if !ok {
			log.Fatalf("Unexpected result type: %s\n", result.Type())
			time.Sleep(SCALE_INTERVAL)
			continue
		}
		log.Printf("Vector length is: %d\n", len(vector))
		if len(vector) == 0 {
			log.Printf("No data returned for query %q", query)
			return
		}

		activeTasksValue := float64(vector[0].Value)

		log.Printf("Current active tasks value is %.2f", activeTasksValue)

		deploymentName := "watcher-bot"
		namespace := "monitoring"

		deployment, err := k8sClient.AppsV1().Deployments(namespace).Get(
			context.Background(),
			deploymentName,
			metav1.GetOptions{},
		)
		if err != nil {
			log.Fatalf("There is some error", err)
		}

		currentReplicas := *deployment.Spec.Replicas
		desiredReplicas := calculateNextReplicas(currentReplicas, activeTasksValue)

		log.Printf("State: Replicas=%d, Desired=%d, Tasks=%.2f (Target TTR=%.1f)",
			currentReplicas, desiredReplicas, activeTasksValue, TARGET_TTR)

		if desiredReplicas != currentReplicas {
			log.Printf("Scaling event needed ")
			deployment.Spec.Replicas = &desiredReplicas

			_, err = k8sClient.AppsV1().Deployments(namespace).Update(
				context.Background(),
				deployment,
				metav1.UpdateOptions{},
			)

			if err != nil {
				log.Printf("Failed to update deployment replicas: %v", err)
			} else {
				log.Printf("Successfully scaled %s to %d replicas.\n", deploymentName, desiredReplicas)
			}

		} else {
			log.Printf("No scaling action required. Replicas are optimal at %d.\n", currentReplicas)
		}

		time.Sleep(SCALE_INTERVAL)
	}

}
