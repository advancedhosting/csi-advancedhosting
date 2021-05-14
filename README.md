# CSI for Advanced Hosting Storage

## How to install

1. Build CSI Driver image:
   ```
   docker build . --build-arg TAG=<IMAGE_TAG>
   ```
2. Create a secret:
   ```
   apiVersion: v1
   kind: Secret
   metadata:
     name: advancedhosting
     namespace: kube-system
   stringData:
     token: "TOKEN"
   ```
3. Deploy CSI. Manifest [**example**](/releases/dev.yaml)
