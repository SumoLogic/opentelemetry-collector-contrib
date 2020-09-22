## <a name="fluentbitk8sprocessor"></a>Fluent bit k8s processor

Processor which convert fluentd tag to the k8s metadata:
 - `pod_name`
 - `namespace`
 - `container_name`
 - `docker_id`

 This plugin is required to correct work of k8s metadata
