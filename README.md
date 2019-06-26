# Consul Connect Escape Hatch Example
This repo shows how to configure the Envoy escape hatch feature of Consul Connect proxies to configure
functionality in Envoy which is not exposed by the Control Plane.

The file consul_config/downstream.hcl is an example service registration which defines the two escape hatch config
elements `envoy_cluster_json` and `envoy_listener_json` which have raw envoy configuration assigned configuring
outlier_detection and retries.

```ruby
service {
  name = "downstream"
  address = "10.5.0.3"
  port= 9000
  connect { 
    sidecar_service {
      port = 20000

  		checks {
        name = "Connect Sidecar Listening"
        tcp = "10.5.0.3:20000"
			  interval = "10s"
      }

      proxy {
				config {}

        upstreams {
          destination_name = "upstream"
          local_bind_port = 9001

          config {
            envoy_cluster_json = <<EOL
              {
                "@type": "type.googleapis.com/envoy.api.v2.Cluster",
                "name": "upstream",
                "type": "EDS",
                "eds_cluster_config": {
                  "eds_config": {
                    "ads": {}
                  }
                },
                "connect_timeout": "5s",
                "outlier_detection": {
                  "consecutive_5xx": 10,
                  "consecutive_gateway_failure": 10,
                  "base_ejection_time": "30s"
                }
              }
            EOL
            
            envoy_listener_json = <<EOL
              {
              "@type": "type.googleapis.com/envoy.api.v2.Listener",
              "name": "upstream:127.0.0.1:9001",
              "address": {
                "socketAddress": {
                  "address": "127.0.0.1",
                  "portValue": 9001
                }
              },
              "filterChains": [
                {
                  "filters": [
                    {
                      "name": "envoy.http_connection_manager",
                      "config": {
                        "stat_prefix": "upstream",
                        "route_config": {
                          "name": "local_route",
                          "virtual_hosts": [
                            {
                              "name": "backend",
                              "domains": ["*"],
                              "routes": [
                                {
                                  "match": {
                                    "prefix": "/"
                                  },
                                  "route": {
                                    "cluster": "upstream",
                                    "timeout": "6s",
                                    "retry_policy": {
                                      "retry_on": "5xx",
                                      "num_retries": 5,
                                      "per_try_timeout": "2s"
                                    }
                                  }
                                }
                              ]
                            }
                          ]
                        },
                        "http_filters": [
                          {
                            "name": "envoy.router",
                            "config": {}
                          }
                        ]
                      }
                    }
                  ]
                }
              ]
            }
            EOL
          }
        }
      }
    } 
  }  
}
```

## Running the demo
To run the demo you require Docker and Docker Compose.  The demo can be run with the command:

```
$ docker-compose up
Starting consul-escape-hatch-demo_downstream_1     ... done
Starting consul-escape-hatch-demo_upstream_1   ... done
Starting consul-escape-hatch-demo_consul_1         ... done
Starting consul-escape-hatch-demo_upstream_proxy_1   ... done
Starting consul-escape-hatch-demo_downstream_envoy_1 ... done
Attaching to consul-escape-hatch-demo_upstream_1, consul-escape-hatch-demo_consul_1, consul-escape-hatch-demo_downstream_1, consul-escape-hatch-demo_upstream_proxy_1,
```

The application can be tested by curling the HTTP endpoint:

```
$ curl localhost:9000
Response request ok from upstream% 
```

The upstream service is designed to randomly error, due to retries in the listener the error should not be returned
to the downstream as this will be internally retried.

```
upstream_1          | Randomly erroring
upstream_1          | Randomly erroring
downstream_1        | 2019/06/26 15:11:39 Calling upstream
upstream_1          | Randomly erroring
upstream_1          | Randomly erroring
downstream_1        | 2019/06/26 15:11:42 Calling upstream
upstream_1          | Randomly erroring
downstream_1        | 2019/06/26 15:11:43 Calling upstream
upstream_1          | Randomly erroring
upstream_1          | Randomly erroring
downstream_1        | 2019/06/26 15:11:44 Calling upstream
```
