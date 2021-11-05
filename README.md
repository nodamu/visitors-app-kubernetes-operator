## Visitors App Operator

* This is an example operator to learn how to build custom controller to manage as stateful application on kubernetes
* Manages a basic React, Django, MySQL app

### Installation
- To run locally (outside a cluster)
```shell
make install run
```
- To build image and push to registry
```shell
make docker-build docker-push IMG=${IMAGE_NAME}:${TAG}

```
- To deploy to a cluster 
```shell
 make deploy
```
- To remove operator from cluster
```shell
make undeploy
```

### Resources
- For more information on operators visit https://sdk.operatorframework.io/
- https://book.kubebuilder.io/
- https://developers.redhat.com/books/kubernetes-operators