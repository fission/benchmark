/*
Copyright 2020 Fission Authors.

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

//Assumptions for this script to work:
// 1. Nodes are tainted with "proxy" as key
// 2. Nginx is deployed with service name of "nginx" as NodePort service

package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	uuid "github.com/satori/go.uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// var name string
var target string
var TTL int64
var weight = int64(1)
var zoneID string

func init() {
	// flag.StringVar(&name, "d", "", "domain name")
	flag.StringVar(&target, "t", "", "target of domain name. The subdomain with which weighted record is to be created")
	flag.StringVar(&zoneID, "z", "", "AWS Zone Id for domain")
	flag.Int64Var(&TTL, "ttl", int64(60), "TTL for DNS Cache")
}

func main() {
	flag.Parse()
	if target == "" || zoneID == "" {
		fmt.Println(fmt.Errorf("incomplete arguments:  t: %s, z: %s", target, zoneID))
		flag.PrintDefaults()
		return
	}
	sess, err := session.NewSession()
	if err != nil {
		fmt.Println("failed to create session,", err)
		return
	}

	svc := route53.New(sess)

	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	externalIP := FindExternalIP(clientset)
	// fmt.Println(externalIP)

	nodeport := FindNginxSvc(clientset)
	// fmt.Println(nodeport)

	weight := FindWeight(len(externalIP))
	// fmt.Println(weight)

	CreateRecords(svc, weight, externalIP)
	endpoint := target + ":" + fmt.Sprint(nodeport)
	fmt.Printf("Service is reachable at %s \n", endpoint)
}

// CreateRecords creates weighted records
func CreateRecords(svc *route53.Route53, weight int, externalIP []string) {
	var cb *route53.ChangeBatch
	var changes []*route53.Change
	var rrslice []*route53.ResourceRecord
	var rrs *route53.ResourceRecordSet
	var rr route53.ResourceRecord

	createAction := route53.ChangeActionCreate
	weight64 := int64(weight)
	recordType := route53.RRTypeA

	for _, ip := range externalIP {

		rr = route53.ResourceRecord{Value: &ip}
		rrslice = append(rrslice, &rr)
		uuid := uuid.NewV4().String()
		rrs = &route53.ResourceRecordSet{
			Name:            &target,
			TTL:             &TTL,
			ResourceRecords: rrslice,
			Type:            &recordType,
			Weight:          &weight64,
			SetIdentifier:   &uuid,
		}

		change := route53.Change{
			Action:            &createAction,
			ResourceRecordSet: rrs,
		}
		changes = append(changes, &change)

		cb = &route53.ChangeBatch{Changes: changes}
		var crsi *route53.ChangeResourceRecordSetsInput
		crsi = &route53.ChangeResourceRecordSetsInput{
			HostedZoneId: &zoneID,
			ChangeBatch:  cb,
		}
		resp, err := svc.ChangeResourceRecordSets(crsi)
		if err != nil {
			fmt.Println(err)
		}
		changes = []*route53.Change{}
		rrslice = []*route53.ResourceRecord{}

		fmt.Println(resp)

	}
}

// FindNginxSvc finds the NginxSvc and returns the port which it is running on
func FindNginxSvc(clientset *kubernetes.Clientset) int32 {
	var nodeport int32
	svcs, err := clientset.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		panic(fmt.Errorf("Error fetching services %v ", err))
	}
	for _, svc := range svcs.Items {
		if svc.Name == "nginx" {
			nodeport = svc.Spec.Ports[0].NodePort
		}
	}
	return nodeport
}

// FindExternalIP gives the external IPs of nodes which are tainted
func FindExternalIP(clientset *kubernetes.Clientset) []string {
	taint := v1.Taint{
		Key: "proxy",
	}

	var nodeAddresses []v1.NodeAddress
	var externalIP []string
	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		panic(fmt.Errorf("Error fetching nodes %v ", err))
	}
	taintedNodes := make([]v1.Node, 0)
	for _, node := range nodes.Items {

		for _, t := range node.Spec.Taints {
			if t.Key == taint.Key {
				taintedNodes = append(taintedNodes, node)
				// 2nd element holds the external ip
				nodeAddresses = append(nodeAddresses, node.Status.Addresses[1])
			}
		}

	}
	for _, address := range nodeAddresses {
		externalIP = append(externalIP, address.Address)
	}
	return externalIP
}

// FindWeight calculates equally distributed weight for each record based on number of nodes nginx is running on
// Ref: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/routing-policy.html
func FindWeight(numOfRecords int) int {
	const max = 255
	weight := max / numOfRecords
	return weight
}
