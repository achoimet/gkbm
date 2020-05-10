/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
package main

import (
	"flag"
	"fmt"
	"github.com/achoimet/gkbm/internal/pkg/k8sAuth"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/pointer"
	"os"
)

const (
	appName = "gkbm"
)

func main() {
	// Subcommands
	aws := flag.NewFlagSet("aws", flag.ExitOnError)
	inCluster := flag.NewFlagSet("inCluster", flag.ExitOnError)
	outCluster := flag.NewFlagSet("outCluster", flag.ExitOnError)

	// aws subcommand flag pointers
	clusterName := aws.String("clusterName", "", "EKS Cluster name. (Required)")
	clusterUrl := aws.String("clusterUrl", "", "EKS Cluster url. (Required)")
	roleArn := aws.String("roleArn", "", "EKS Role to assume. (Required)")
	undoAws := aws.Bool("undo", false, "undo the deployment")
	undoInCluster := inCluster.Bool("undo", false, "undo the deployment")
	undoOutCluster := outCluster.Bool("undo", false, "undo the deployment")
	// Verify that a subcommand has been provided
	// os.Arg[0] is the main command
	// os.Arg[1] will be the subcommand
	if len(os.Args) < 2 {
		fmt.Println("aws or inCluster or outCluster subcommand is required")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "aws":
		err := aws.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("failed to parse subcommand arguments: %v", err)
			os.Exit(1)
		}
	case "inCluster":
		err := inCluster.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("failed to parse subcommand arguments: %v", err)
			os.Exit(1)
		}
	case "outCluster":
		err := outCluster.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("failed to parse subcommand arguments: %v", err)
			os.Exit(1)
		}
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Check which subcommand was Parsed using the FlagSet.Parsed() function. Handle each case accordingly.
	// FlagSet.Parse() will evaluate to false if no flags were parsed (i.e. the user did not provide any flags)
	var clientSet *kubernetes.Clientset
	var err error
	if aws.Parsed() {
		flag.Parse()
		// Required Flags
		if *clusterName == "" || *clusterUrl == "" || *roleArn == "" {
			aws.PrintDefaults()
			os.Exit(1)
		}
		//Create aws session
		sess := session.Must(session.NewSession(&awssdk.Config{
			Region: awssdk.String("eu-west-3"),
		}))

		clientSet, err = k8sAuth.AuthenticateToEks(*clusterName, *clusterUrl, *roleArn, sess)
		if err != nil {
			fmt.Printf("failed to authenticate: %v", err)
			os.Exit(1)
		}
	}
	if outCluster.Parsed() {
		flag.Parse()
		clientSet, err = k8sAuth.AuthenticateOutOfCluster()
		if err != nil {
			fmt.Printf("failed to authenticate: %v", err)
			os.Exit(1)
		}
	}
	if inCluster.Parsed() {
		flag.Parse()
		clientSet, err = k8sAuth.AuthenticateInCluster()
		if err != nil {
			fmt.Printf("failed to authenticate: %v", err)
			os.Exit(1)
		}
	}
	if *undoAws || *undoOutCluster || *undoInCluster {
		err = clientSet.CoreV1().Namespaces().Delete(appName, &metav1.DeleteOptions{})
		if err != nil && errors.ReasonForError(err) != metav1.StatusReasonNotFound {
			fmt.Printf("failed to undo: %v", err)
			os.Exit(1)
		}
		fmt.Println("App deleted.")
	} else {
		err = deploy(clientSet)
		if err != nil {
			fmt.Printf("failed to deploy: %v", err)
			os.Exit(1)
		}
	}
}

func deploy(client *kubernetes.Clientset) error {
	myNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: appName},
	}
	_, err := client.CoreV1().Namespaces().Create(myNamespace)
	if err != nil && errors.ReasonForError(err) != metav1.StatusReasonAlreadyExists {
		return err
	}
	err = createDeployment(client.AppsV1())
	if err != nil {
		return err
	}
	err = createService(client.CoreV1())
	if err != nil {
		return err
	}
	fmt.Println("App deployed.")
	return err
}

func createDeployment(clientAppsV1 typedappsv1.AppsV1Interface) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
			Labels: map[string]string{
				"app": appName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app": appName,
			}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: appName,
					Labels: map[string]string{
						"app": appName,
					},
				},
				Spec: corev1.PodSpec{Containers: []corev1.Container{
					{
						Name:            appName,
						Image:           "achoimet/gkbm",
						Ports:           []corev1.ContainerPort{{Name: "http", ContainerPort: 3000}},
						ImagePullPolicy: corev1.PullAlways,
					},
				}},
			},
		},
	}

	_, err := clientAppsV1.Deployments(appName).Create(deployment)
	if err != nil {
		return err
	}
	return err
}

func createService(clientCoreV1 typedcorev1.CoreV1Interface) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
			Labels: map[string]string{
				"app": appName,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					TargetPort: intstr.IntOrString{IntVal: 3000},
					Port:       80,
				},
			},
			Selector: map[string]string{
				"app": appName,
			},
			Type: corev1.ServiceTypeLoadBalancer,
		},
	}
	_, err := clientCoreV1.Services(appName).Create(service)
	if err != nil {
		return err
	}
	return err
}
