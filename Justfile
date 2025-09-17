build:
    docker build -t ghcr.io/cofide/mtls-secret-updater:latest .

load: build
    kind load docker-image ghcr.io/cofide/mtls-secret-updater:latest --name connect

deploy: load
    just undeploy
    kubectl create --context kind-connect -f deployment/manifests/test-pod.yaml

undeploy:
    kubectl delete --context kind-connect -f deployment/manifests/test-pod.yaml || true

test-release:
    goreleaser release --snapshot --clean
