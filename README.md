This package is used in conjunction with socketmaster to provide graceful restart.

Genesis
-------

Its not easy to fork and do graceful restarts in golang, the same way they are done in C. There is no equivalent for the fork syscall.

While there are several alternative strategies, such as facebooks grace, they suffer from some limitations. Primary among those is that the new process is not related to the old. This means if you are using a monitor like upstart or monit, they lose the ability to track the process status (unless you use a pid file). This affects ability to restart on crash, but also other things such as logrotation.

Our solution to this is to

- use a C binary named [socketmaster](https://github.com/zimbatm/socketmaster). socketmaster binds to the listening socket, with the -listen tcp://<port> directive.
- socketmaster starts the go program (-command <pathtobinary>) and passes the socket to it.
- when the go program exits, socketmaster starts a new instance, and passes the socket to the new program

The grace library makes this transparent. Generally, in your go program, all that you need to do is to import grace, and then use grace.Serve instead of http.Serve

Example
-------

Sample Upstart config with [socketmaster](https://github.com/zimbatm/socketmaster)

```
start on (net-device-up
          and local-filesystems
          and runlevel [2345])
stop on runlevel [016]

respawn
respawn limit 5 60

limit nofile 4096 4096

exec socketmaster -command=/usr/bin/goapp -listen tcp://:9000 -- -l /var/log/goapp/goapp.access.log -e /var/log/goapp/goapp.error.log >> /var/log/goapp/goapp.upstart.log
```

Sample Go app

```
import (
  "log"
  "net/http"
  grace "github.com/b3bas/grace"
)

func main() {
  // install handlers
  http.Handle("/foo/bar", fooBarHandler)
  log.Fatal(grace.Serve(":9000", nil))
}
```
