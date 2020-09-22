# Fluent-bit k8s processor

Processor which converts fluentd tags to the following k8s metadata:

- `pod_name`
- `namespace`
- `container_name`
- `docker_id`

 This plugin is required for correct work of k8s metadata
