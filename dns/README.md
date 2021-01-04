# AWS Route53 

## What does this program do?
It detects the number of tainted nodes on the cluster on which Nginx is deployed , calculates weight such that traffic is equally distributed to all the records created and creates the `A` record with desired subdomain name.

## Steps to be followed
- Nodes are tainted with label `proxy` so that only nginx pods are scheduled on these nodes. For example:

    `kubectl taint node ip-192-168-7-148.ap-south-1.compute.internal   proxy=:NoExecute `

- Deploy Fission

- Deploy Nginx along with configuration which can resolve to Fission Router. 

    `kubectl apply -f nginx/`

- Create a hosted zone in Route53 

- Run the program with following parameters: 

    - Target: Subdomain to be created in the hosted zone 
    - ZoneID : The zone ID of the hosted
    - TTL(optional) :  cache time to live in seconds :

`go run main.go -t test1.infracloud.club -z Z04819012F7MC0LH31Q6R`


