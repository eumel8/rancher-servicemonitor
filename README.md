# Rancher ServiceMonitor

Collect metrics from Rancher API

- management.cattle.io

## installation

via Helm chart

- create an admin token (without scope) on your Rancher server to monitor

- create a secret in the target namespace of the app (must not be the Rancher upstream cluster)


```bash
$ kubectl create secret generic ranchertoken --from-literal RANCHER_TOKEN=token-xxxx
```

- install the helm chart, adjust at least the `RANCHER_URL` env var

```bash
$ helm upgrade -i rancher-servicemonitor ./chart
```

## dev testing

```bash
go test -coverprofile=c.out ./...
go tool cover -html=c.out 
````
