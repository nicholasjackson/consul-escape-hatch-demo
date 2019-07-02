# Consul Connect Escape Hatch Example
This repo shows how to configure the Envoy escape hatch feature of Consul Connect proxies to configure
functionality in Envoy which is not exposed by the Control Plane.

The file consul_config/downstream.hcl is an example service registration which defines the two escape hatch config
elements `envoy_cluster_json` and `envoy_listener_json` which have raw envoy configuration assigned configuring
outlier_detection and retries.

Outlier detection is configured for the local cluster which are proxied requests to from the data plane to the local
service instance. `envoy_local_cluster_json` is an override for the default configuration and circuit breaking with 
a limit of 1 concurrent request has been configured.

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
				config {
          envoy_local_cluster_json = <<EOL
           {
             "@type": "type.googleapis.com/envoy.api.v2.Cluster",
             "name": "local_app",
             "connect_timeout": "5s",
             "circuit_breakers": {
               "thresholds": [
                 {
                   "priority": "HIGH",
                   "max_requests": 1
                 }
               ]
             },
             "load_assignment": {
              "cluster_name": "local_app",
              "endpoints": [
               {
                "lb_endpoints": [
                 {
                  "endpoint": {
                   "address": {
                    "socket_address": {
                     "address": "127.0.0.1",
                     "port_value": 9000
                    }
                   }
                  }
                 }
                ]
               }
              ]
             }
           }
        EOL
        }

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

## Circuit breaking
Envoy has also been configured with circuit breaking using the escape hatch capability which allows you to override
the automatically generated cluster configuration for the local service. Looking at the following example you will see
that the downstream service has the `envoy_local_cluster_json` override set to an Envoy cluster configuration block.
This block defines circuit_breakers and set the max_requests to 10.
[https://www.envoyproxy.io/learn/circuit-breaking](https://www.envoyproxy.io/learn/circuit-breaking)

```
proxy {
	config {
    envoy_local_cluster_json = <<EOL
     {
       "@type": "type.googleapis.com/envoy.api.v2.Cluster",
       "name": "local_app",
       "connect_timeout": "5s",
       "circuit_breakers": {
         "thresholds": [
           {
             "priority": "HIGH",
             "max_requests": 10
           }
         ]
       },
       "load_assignment": {
        "cluster_name": "local_app",
        "endpoints": [
         {
          "lb_endpoints": [
           {
            "endpoint": {
             "address": {
              "socket_address": {
               "address": "127.0.0.1",
               "port_value": 9000
              }
             }
            }
           }
          ]
         }
        ]
       }
     }
  EOL
  
  }
```

If you run Apache bench with 5 concurrent requests you will see that the majority of requests are succeeding as the
circuit breaker is not tripping (concurrency less than configured max_requests).

```
$ ab -n 1000 -c 5 http://localhost:9000/
This is ApacheBench, Version 2.3 <$Revision: 1826891 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking localhost (be patient)
Completed 100 requests
...
Completed 1000 requests
Finished 1000 requests


Server Software:
Server Hostname:        localhost
Server Port:            9000

Document Path:          /
Document Length:        33 bytes

Concurrency Level:      5
Time taken for tests:   21.624 seconds
Complete requests:      1000
Failed requests:        1
   (Connect: 0, Receive: 0, Length: 1, Exceptions: 0)
Total transferred:      149997 bytes
HTML transferred:       32997 bytes
Requests per second:    46.25 [#/sec] (mean)
Time per request:       108.119 [ms] (mean)
Time per request:       21.624 [ms] (mean, across all concurrent requests)
Transfer rate:          6.77 [Kbytes/sec] received
```

If you now set the `max_requests` to `1`:


```
proxy {
	config {
    envoy_local_cluster_json = <<EOL
     {
       "@type": "type.googleapis.com/envoy.api.v2.Cluster",
       "name": "local_app",
       "connect_timeout": "5s",
       "circuit_breakers": {
         "thresholds": [
           {
             "priority": "HIGH",
             "max_requests": 1
           }
         ]
       },
       "load_assignment": {
        "cluster_name": "local_app",
        "endpoints": [
         {
          "lb_endpoints": [
           {
            "endpoint": {
             "address": {
              "socket_address": {
               "address": "127.0.0.1",
               "port_value": 9000
              }
             }
            }
           }
          ]
         }
        ]
       }
     }
  EOL
  
  }
```

And again run the test:

```
$ ab -n 1000 -c 5 http://localhost:9000/
This is ApacheBench, Version 2.3 <$Revision: 1826891 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking localhost (be patient)
Completed 100 requests
...
Completed 1000 requests
Finished 1000 requests


Server Software:
Server Hostname:        localhost
Server Port:            9000

Document Path:          /
Document Length:        30 bytes

Concurrency Level:      5
Time taken for tests:   16.304 seconds
Complete requests:      1000
Failed requests:        735
   (Connect: 0, Receive: 0, Length: 735, Exceptions: 0)
Total transferred:      149205 bytes
HTML transferred:       32205 bytes
Requests per second:    61.33 [#/sec] (mean)
Time per request:       81.522 [ms] (mean)
Time per request:       16.304 [ms] (mean, across all concurrent requests)
Transfer rate:          8.94 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.1      0       1
Processing:     3   80  47.7    104     598
Waiting:        3   80  47.8    104     598
Total:          3   81  47.7    104     598

Percentage of the requests served within a certain time (ms)
  50%    104
  66%    105
  75%    105
  80%    106
  90%    107
  95%    118
  98%    126
  99%    138
 100%    598 (longest request)
```

You will see that there are large number of failed requests, this is because the circuit breaker in Envoy is tripping
as the max_requests is now less that the concurrency in ApacheBench.

### Note
When changing the configuration values for the Consul config, always restart docker compose with `docker-compose rm`.
Compose will occasionally cache the container and will not pickup the change to the config file.

