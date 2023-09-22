# counter
Counter example deployment using Kubeedge with LCD Display. Steps are down below:

First step:

Connect the LCD Dsplay 1602 to the raspberry pi (Raspberry pi 4 was used for this demo).

Second step:

After create the cluster and adding the edge node, apply manifest files in /crds

kubectl create -f kubeedge-counter-model.yaml

kubectl create -f kubeedge-counter-instance.yaml #set your edge bode name accordingly

kubectl create -f kubeedge-web-controller-app.yaml

kubectl create -f ctr_deployment.yaml  

# Check the web app using the master_node_ip:80 and control the counter

Use this command to watch status from cloud: kubectl get device counter -o yaml -w
