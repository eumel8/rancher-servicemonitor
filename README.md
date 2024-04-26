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
```

ref:
- https://github.com/stretchr/testify?tab=readme-ov-file#suite-package
- https://github.com/rancher/rancher/blob/9c4b89a0a2e11d6037bd0760d80560abf3e65d53/tests/v2/integration/projects/project_user_test.go
- https://github.com/rancher/cluster-api/blob/bc756c4e7ed08313413c1300859bd4dceeccb25a/docs/book/src/developer/testing.md
