[![Build Status](https://travis-ci.org/eBay/collectbeat.svg?branch=master)](https://travis-ci.org/eBay/collectbeat)
# collectbeat


Collectbeat provides discovery capabilities on top of Filebeat/Metricbeat. 

### Build
To build collectbeat just run:

```
make
```

### Run
To run collectbeat in filebeat mode:

`CLUSTER=minikube NODE=minikube NAMESPACE=default ./collectbeat filebeat -e -v`

To run collectbeat in metricbeat mode:
`CLUSTER=minikube NODE=minikube NAMESPACE=default `./collectbeat metricbeat -e -v`

All flags that are supported by filebeat and metricbeat are supported out of the box when running collectbeat in either filebeat or metricbeat mode respectively. The only piece of configuration that varies from stock filebeat and metricbeat is the `discovery` section. All other configuration can be done similar to how Beats documents it.

## Discovery

Discovery is the process of identifying workloads from which logs and metrics can be collected from. Discovery is currently available for the following:

* Kubernetes
* Docker (coming soon)


### Kubernetes

Kubernetes empowers customers to drop a Docker container as a Pod and let
Kubernetes manage the lifecycle of the application. If the Pod dies, then Kubernetes takes care of bringing it up. If it needs to be scaled out automatically because of CPU, Memory parameters then Kubernetes takes care of that as well. Similarly, Docker provides the contract that if an application logs to `stdout` then Docker collects it in log files. As part of the Kubernetes experience we want to be able to provide first class experience with logs and metrics. We want to keep the experience simple and convenient for the users to be able to log and generate metrics which we can collect and ship.

#### Application Logs

Docker provides the ability to users to write their logs on to `stdout`/`stderr`and the logs get automatically collected in the host. Similarly in Kubernetes
we want to provide a simple way for users to write application logs. If the user writes their logs on to `stdout`/`stderr` then we collect the logs and make sure that the logs are annotated properly with the following information:

1.  Namespace

2.  Pod Name

3.  Pod Labels

4.  Container Name

5.  Application Name

Log lines such as stack traces are more than one log line that are stitched together. The log collection system can not identify how to stitch stack traces unless users provide sufficient knowledge about the same. This capability is called `multiline`. Users can provide multiline configurations using the following annotations:

`io.collectbeat.logs/pattern=<pattern>` would provide a pattern for all containers in the Pod. The pattern is specified in [Golang regexp](https://golang.org/pkg/regexp/syntax/) format. By default the user provides the pattern by which the trailing lines of the stack trace would match.

For example, consider the following stack trace:

```
Exception in thread "main" java.lang.NullPointerException
        at com.example.myproject.Book.getTitle(Book.java:16)
        at com.example.myproject.Author.getBookTitles(Author.java:25)
        at com.example.myproject.Bootstrap.main(Bootstrap.java:14)
```

This would be matched by the annotation `io.collectbeat.logs/pattern="^[[:space:]]"`

If the user wishes to match the first line of the stack trace it can be done as done for the following example:

```
[2015-08-24 11:49:14,389][INFO ][env                      ] [Letha] using [1] data paths, mounts [[/
(/dev/disk1)]], net usable_space [34.5gb], net total_space [118.9gb], types [hfs]
```

```
io.collectbeat.logs/pattern: "^\[[0-9]{4}-[0-9]{2}-[0-9]{2}"
io.collectbeat.logs/negate: true
io.collectbeat.logs/match: after
```

There can be scenarios where users have more than one container that require specific multiline patterns. Configurations for individual containers can be provided by appending the container name to the annotation as follows:

```
io.collectbeat.logs.container1/pattern: "^\[[0-9]{4}-[0-9]{2}-[0-9]{2}"
io.collectbeat.logs.container1/negate: true
io.collectbeat.logs.container1/match: after
```

The signal containing the log is quite verbose and contains all the
metadata associated with the application that had generated logs. Logs can have more information than just some arbitrary text and could be parsed to extract out the information. 

#### Application Metrics

Since Kubernetes is a truly multi-tenanted environment where users can drop any kind of application as a docker container into Kubernetes. Applications could potentially report metrics in their own customized way. In such a multi-tenanted environment it is a challenge to collect these metrics in a standard way. Hence we expect the users to “annotate” their pod specification with some basic information so that we can identify the tenant, understand what kind of metrics are exposed and how best to collect them.

The information that is expected in the Pod [*annotations*](http://kubernetes.io/docs/user-guide/annotations/) are:


  Name | Mandatory | Default Value | Description
  --- | --- | --- | ---
  `io.collectbeat.metrics/type` | Yes|  | What the format of the metrics being exposed is. Ex: `prometheus`, `dropwizard`, `mongodb`
  `io.collectbeat.metrics/endpoints` | Yes | | Comma separated locations to query the metrics from. Ex: `":8080/metrics, :9090/metrics"`
  `io.collectbeat.metrics/interval` | No | 1m | Time interval for metrics to be polled. Ex: `10m`, `1m`, `10s`
  `io.collectbeat.metrics/timeout` | No | 3s | Timeout duration for polling metrics. Ex: `10s`, `1m`
`io.collectbeat.metrics/namespace` | No | | Namespace to be provided for Dropwizard/Prometheus/HTTP metricsets.


To provide more clarity on the above fields, we collect metrics using
collectbeat which is built on top of the [*Beats*](https://www.elastic.coproducts/beats) framework. In order to effectively collect the metrics from user applications we require the two mandatory fields which are the type and the endpoints. Metric type is nothing but a module in beats that can understand how to make sense out of the metrics that are being exposed. For example if one considers the “mysql” metric type, the module understands how to use the endpoint which would be host:3306 and query the mysql for the application metrics. Once these metrics are collected the filters are applied based on what the user provides as to include and exclude and the resultant set is shipped to the configured backend.

If a module requires additional fields apart from the above fields then those fields can be added as “metric.fieldname” in the annotations. For example: the redis module requires network as an additional field. So, the payload can be annotated as “metric.network” and the value can be provided.

##### List of supported metric modules:

-   [Prometheus](https://www.elastic.co/guide/en/beats/metricbeat/master/metricbeat-module-prometheus.html)

-   [Dropwizard](https://www.elastic.co/guide/en/beats/metricbeat/master/metricbeat-module-dropwizard.html)

-   [Metricbeat supported modules](https://www.elastic.co/guide/en/beats/metricbeat/master/metricbeat-modules.html)

Once the Pod spec has been annotated and the Pod is deployed, we take
care of collecting metrics from the specified endpoint and annotate them
with the following information and send it to the configured backend:

1.  Namespace

2.  Pod Name

3.  Pod Labels

4.  Container Name

5.  Application Name


##### What if my application is not listed in the supported metric modules ?

The prometheus module is designed in such a way that it can query both applications that are instrumented and applications that use “[*prometheus exporter*](https://prometheus.io/docs/instrumenting/exporters/)”. The documentation on prometheus exporters covers the list of all supported applications that have prometheus exporters written for them. If your application has a supported prometheus exporter then it can be added to either your docker container itself or as a [*sidecar*](http://blog.kubernetes.io/2015/06/the-distributed-system-toolkit-patterns.html)
to your pod. Once that is done, the pod can be annotated with metric.type=prometheus and `io.collectbeat.metrics/endpoints=":port_at_which_exporter_runs/path"`

With the side car approach, the user needs to be aware that the total memory and CPU that the user allocates to the pod is the memory and CPU that is available to the main application container and the sidecar container. The advantage of the side car approach is that the exporter is packaged in neatly as a separate container and can be upgraded by the user without having to disturb the user application.

A list of prometheus exporters can be found here:

[*https://prometheus.io/docs/instrumenting/exporters/*](https://prometheus.io/docs/instrumenting/exporters/)

It is entirely in the user’s discretion to either use a side car approach or to package the exporter inside of the application container itself. 


##### What if I want to push metrics instead of exposing endpoints?
The metrics collection platform provides two mechanisms to push metrics. They are:

###### Push metrics via Graphite Protocol
There are a variety of applications that push metrics via [Graphite Protocol](http://graphite.readthedocs.io/en/latest/feeding-carbon.html) like [CollectD](https://collectd.org) and [StatsD](https://github.com/etsy/statsd). Beats accepts graphite protocol messages that are written to a well known port in `UDP` protocol. This however requires some configuration to be present on the PodSpec and the CollectD/StatsD container.

If a user is using CollectD, the graphite output plugin needs to be configured as follows:

```
LoadPlugin "write_graphite"
<Plugin "write_graphite">
 <Node "kubernetes host">
   Host "$NODE_NAME"
   Port "50040"
   Prefix "foo.$NAMESPACE.$POD."
   Protocol "udp"
   SeparateInstances true
   StoreRates false
   AlwaysAppendDS false
 </Node>
</Plugin>
``` 

CollectD doesn't support replacement of environment variables, hence it has to be done by a pre-run shell script. A sample container can be found in the appendix. The following annotations would also be required on the PodSpec for metrics to be meaningfully parsed.

  Name | Mandatory | Description
  --- | --- | ---
  `io.collectbeat.graphite/filter` | Yes| The Filter to be applied to identify which workload's metrics are being processed.
  `io.collectbeat.graphite/template` | Yes | The template based on which the metric is parsed to split dimensions and the metric name.
  `io.collectbeat.graphite/namespace` | Yes | The namespace into which the metric is persisted.
  `io.collectbeat.graphite/tags` | No | Comma separated key/vvalue pairs that are written in as dimensions in the end metric.

Few examples of templates and filters can be found below:

```
filter: foo.*
template: .pod.namespace.host.host.host.host.metric*

This would parse the following inputs as follows:
foo.p1.ns1.a.lvs.ebay.com.total_count -> pod=p1, namespace=ns1, host=a.lvs.ebay.com, metricName=total_count

filter: foo.bar.*
template: .dim1.namespace.pod.metric*

This would parse the following input as:
foo.bar.p1.ns1.cpu.max.usage -> dim1=bar, pod=p1, namespace=ns1, metricName=cpu.max.usage 
```

### Appendix:

**Sample Deployment that has metrics collected:**

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: prometheus
spec:
  replicas: 3
  template:
    metadata:
      labels:
        app: prometheus
      annotations:
        io.collectbeat.metrics/type: "prometheus"
        io.collectbeat.metrics/endpoints: ":9090"
        io.collectbeat.metrics/namespace: "prom"
        io.collectbeat.metrics/interval: "1m"
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus
        ports:
        - containerPort: 9090
        resources:
          limits:
            cpu: 100m
            memory: 200Mi
          requests:
            cpu: 100m
            memory: 200Mi

```

# Maintainers

* [Vijay Samuel](https://github.com/vjsamuel) (Twitter: [@vjsamuel_](http://twitter.com/vjsamuel_))
