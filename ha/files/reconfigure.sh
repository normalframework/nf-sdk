#!/bin/bash

if [ $# -eq 0 ]; then
    TO_IP=$(docker exec nf_redis-sentinel_1 redis-cli -p 26379 SENTINEL GET-MASTER-ADDR-BY-NAME nfmaster | head -n1)
else
    TO_IP=$6
fi
echo "Client reconfiguration; new master is $TO_IP"

if [ "$TO_IP" = "{{ ansible_host }}" ]; then
    run-parts /etc/nf/promote.d/ --regex '^.*$'
else
    run-parts /etc/nf/demote.d/ --regex '^.*$'
fi

exit 0
