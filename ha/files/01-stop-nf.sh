#!/bin/sh

echo "Stopping nf"
curl -XPOST --unix-socket /var/run/docker.sock http:/localhost/containers/nf_nf_1/stop -o-

