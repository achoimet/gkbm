package kubernetes

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

func AuthenticateToEks(clusterName string, cluster *api.Cluster, roleArn string, session *session.Session) (clientcmd.ClientConfig, error) {

	clusters := make(map[string]*api.Cluster)
	clusters[clusterName] = cluster
	c := &api.Config{Clusters: clusters}

	g, err := token.NewGenerator(true, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create iam-authenticator token generator: %v", err)
	}

	t, err := g.GetWithRoleForSession("eks_test", roleArn, session)
	if err != nil {
		return nil, fmt.Errorf("failed to get token for eks: %v", err)
	}
	clientConfig := clientcmd.NewDefaultClientConfig(*c, &clientcmd.ConfigOverrides{Context: api.Context{Cluster: clusterName}, AuthInfo: api.AuthInfo{Token: t.Token}})

	return clientConfig, nil
}
