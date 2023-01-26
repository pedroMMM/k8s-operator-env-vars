build: 
	minikube image build -t echosvr:dev .

run: build
	kubectl apply -f ./k8s.yml
