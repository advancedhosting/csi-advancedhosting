# CSI for Advanced Hosting Storage

## How to install


1. Create a secret:
   ```
   apiVersion: v1
   kind: Secret
   metadata:
     name: advancedhosting
     namespace: kube-system
   stringData:
     token: "TOKEN"
   ```
2. Deploy CSI:
   ```
   kubectl apply -f https://raw.githubusercontent.com/advancedhosting/csi-advancedhosting/master/deploy/advancedhosting-csi-{VERSION}.yaml
   ```
