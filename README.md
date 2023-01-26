# k8s-operator-env-vars
Injects Environment Variables into Deployments.

## Notes 
- Using 'minikube' for portability.
- Using the 3rd party 'Operator Framework' over the official 'kubebuilder'. Because it builds QoL on top of 'kubebuilder' primitives.
- When it came to the custom resource the Operator should use, I decided to be a bad neighbor for the sake of simplicity and attached the Operator to the existing core resources.
- I attempted to use the new generic slice functions but quickly realized that I need more practice with it and moved on in the interest of time. - https://pkg.go.dev/golang.org/x/exp/slices
- Between interviews and dinner breaks, my minikube went belly up. While waiting for the "cluster" restart I realized I missed some requeirements! I went the full Operator direction while I should had gone with a simple Microservice. This is specially bad because more than half of the time I sent on this was spent bending the Operator SDK primitives to my will.
- At the end of the day the Operator Framework and I failed to set the right permissions. I have followed the permission path several times and I can't see anything wrong. It's one of those times for a 2nd pair of eyes. 😭

## How-to
Setup test bed
```bash
minikube start
```

Setup test Deployment and ConfigMap
```bash
cd testing
make run
```

Setup Operator
```bash
minikube image build -t controller:latest .
make deploy
```

Wipe test bed
```bash
minikube delete
```
