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
	"github.com/achoimet/gkbm/internal/kubernetes"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	v14 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes2 "k8s.io/client-go/kubernetes"
	v13 "k8s.io/client-go/kubernetes/typed/apps/v1"
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
			panic(err)
		}
	case "inCluster":
		err := inCluster.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}
	case "outCluster":
		err := outCluster.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Check which subcommand was Parsed using the FlagSet.Parsed() function. Handle each case accordingly.
	// FlagSet.Parse() will evaluate to false if no flags were parsed (i.e. the user did not provide any flags)
	var clientSet *kubernetes2.Clientset
	var err error
	if aws.Parsed() {
		// Required Flags
		if *clusterName == "" || *clusterUrl == "" || *roleArn == "" {
			aws.PrintDefaults()
			os.Exit(1)
		}
		//Create aws session
		sess := session.Must(session.NewSession(&awssdk.Config{
			Region: awssdk.String("eu-west-3"),
		}))

		clientSet, err = kubernetes.AuthenticateToEks(*clusterName, *clusterUrl, *roleArn, sess)
		if err != nil {
			panic(err)
		}

	}
	if outCluster.Parsed() {
		clientSet, err = kubernetes.AuthenticateOutOfCluster()
	}
	if inCluster.Parsed() {
		clientSet, err = kubernetes.AuthenticateInCluster()
	}
	err = deployMe(clientSet)
	if err != nil {
		panic(err)
	}

}

func deployMe(client *kubernetes2.Clientset) error {
	myNamespace := &v12.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: appName},
	}
	_, err := client.CoreV1().Namespaces().Create(myNamespace)
	if err != nil && errors.ReasonForError(err) != v1.StatusReasonAlreadyExists {
		return err
	}
	err = createDeployment(client.AppsV1())
	if err != nil {
		return err
	}
	fmt.Println("App deployed.")
	return err
}

func createDeployment(clientAppsV1 v13.AppsV1Interface) error {
	deployment := &v14.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name: appName,
			Labels: map[string]string{
				"app": appName,
			},
		},
		Spec: v14.DeploymentSpec{
			Replicas: pointer.Int32Ptr(2),
			Selector: &v1.LabelSelector{MatchLabels: map[string]string{
				"app": appName,
			}},
			Template: v12.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Name: appName,
					Labels: map[string]string{
						"app": appName,
					},
				},
				Spec: v12.PodSpec{Containers: []v12.Container{
					{
						Name:  appName,
						Image: "achoimet/gkbm",
						Ports: []v12.ContainerPort{{Name: "http", ContainerPort: 3000}},
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
