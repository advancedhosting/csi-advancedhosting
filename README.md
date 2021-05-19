# CSI for Advanced Hosting Storage

## How to install


Create a secret:
```
apiVersion: v1
kind: Secret
metadata:
  name: advancedhosting
  namespace: kube-system
stringData:
  token: "TOKEN"
```
Deploy CSI via kubectl:
```
kubectl apply -f https://raw.githubusercontent.com/advancedhosting/csi-advancedhosting/master/deploy/advancedhosting-csi-{VERSION}.yaml
```

Deploy CSI via helm:
```
helm repo add ah-csi https://advancedhosting.github.io/csi-advancedhosting
helm repo update
helm install ah-csi/ah-csi-driver
```