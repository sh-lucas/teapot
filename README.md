# **teapot**
  A simple cli wrapper that gets all the output from your application into a buffer and sends every 100 lines to your private deployment url.    
  Made for when Promtail is overkill or the Sidecar config is not accessible, and you still have work to do.    
  It's simple, efficient and (relatively) secure, but **not intended for production deployments**.    
  It's **absolutely necessary to have https**. The cli will crash on boot if you try http.    
  > I recommend cloudflare tunnels for a easy-to-setup, secure and reliable https endpoint.     
  
  ****:
  - Basic Auth: you set up a key for reading and a key for writing logs. Simpler is safer.    
  - Asyncysh: asyncronous, non-blocking loop; might lose some logs if network is down, but your app runs smoothly.    
  - Statelessly: not really; but restarting either the containers is usually ok.    


## Sobre o framework/mug (home-brewed)
- You should `go install github.com/sh-lucas/mug@latest`.
- Run `mug gen` to generate the glue files.
- The directives `// mug:handler` declares chi endpoints and common middlewares.
- dotenv is unecessary because mug generates `./cup/envs.go`; good hacking `=)`
