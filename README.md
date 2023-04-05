NF SDK
=====

Welcome to the NF SDK.  This repository contains a `docker-compose`
file which you can use to run a local copy of NF; and generated client
libraries for Javascript and Python for the gRPC APIs.  You can also
use the REST API directly.

Installation Instructions: Ubuntu
-------------------------

First, install Docker
log into the Azure ACR repository using the credentials you obtained from Normal:

```
$ sudo apt install docker docker-compose git
$ sudo docker login -u <username> -p <key> normalframework.azurecr.io
```

After that, clone this repository:

```
$ git clone https://github.com/normalframework/nf-sdk.git
$ cd nf-sdk
```

Finally, pull the containers and start NF:
```
$ sudo docker-compose pull
$ sudo docker-compose up -d
```

After this step is finished, you should be able to visit the
management console at [http://localhost:8080](http://localhost:8080)
on that machine.
