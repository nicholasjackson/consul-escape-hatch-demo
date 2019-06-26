service {
  name = "upstream"
  address = "10.5.0.4"
  port= 9001
  connect { 
    sidecar_service {
      port = 20000
  		
			check {
        name = "Connect Sidecar Listening"
        tcp = "10.5.0.4:20000"
				interval = "10s"
      }
     
			proxy {
				config {
				}
			}
    } 
  }  
}
