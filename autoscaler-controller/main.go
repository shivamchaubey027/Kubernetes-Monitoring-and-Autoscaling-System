package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/common/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const prometheusURL = "http://localhost:9090"

func main() {
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		log.Fatalf("Error creating Prometheus client: %v", err)
	}
	log.Printf("Client Connected")
	v1api := v1.NewAPI(client)
	fmt.Printf("Successfully created Prometheus client connecting to %s\n", prometheusURL)
	query := "watcherBot_active_tasks"
	result, warnings, err := v1api.Query(context.Background(), query, time.Now())

	if err != nil {
		log.Fatalf("Error querying Prometheus %v\n", err)
	}

	if len(warnings) > 0 {
		log.Printf("Prometheus query returned warnings %v\n", warnings)
	}

	fmt.Printf("Query successful. Result Type: %T\n", result)

	vector, ok := result.(model.Vector)
	if !ok {
		log.Fatalf("Unexpected result type: %s\n", result.Type())
	}
	log.Printf("Vector length is: %d\n", len(vector))
	if len(vector) == 0 {
		log.Printf("No data returned for query %q", query)
		return
	}

	activeTasksValue := float64(vector[0].Value)

	log.Printf("Current active tasks value is %.2f", activeTasksValue)

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed in get in-cluster config: %v", err)
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Clientset: %v", err)
	}
	log.Println("Client Created")

	deploymentName := "watcher-bot"
	namespace := "monitoring"

	deployment, err := clientSet.AppsV1().Deployments(namespace).Get(
		context.Background(),
		deploymentName,
		metav1.GetOptions{},
	)
	if err != nil {
		log.Fatalf("There is some error", err)
	}
	log.Println("Current replicas for %s : %d", deploymentName, *&deployment.Spec.Replicas)
}
