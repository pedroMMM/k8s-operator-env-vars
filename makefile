
k8s-start:
	minikube start
	kubectl cluster-info

k8s-end:
	minicuke stop

build-echosvr:
	minikube image build -t echosvr:dev -f ./echosvr/dockerfile .

build: build-echosvr

run: run-echosvr

run-echosvr:
	kubectl apply -f ./echosvr/k8s.yml