
High Availability Playbooks
===========================

These instructions are tested on Ubuntu Server 22; but should be able
to be easily adapted to other environments.  They are desgined to
automate some of the more common administration tasks on a NF/HA
cluster.

1. Make sure `/var/nf` has been configured as a shared network mount
   on all nodes.

2. Edit hosts.yml to provide the correct IP addresses and usernames to
   communicate with the NF nodes.  Do not rename the nodes since they
   are used in configuration templates.

3. Execute `ansible-playbook -i hosts.yml 00-deploy-system-dependencies.yml --ask-become-pass`
   if needed to install system dependencies like docker and the required
   python mdules for ansible.

4. Edit the volumes section of `docker-compose-1.5-redis-ha.yaml.j2`
   as needed to place the Redis volumes on each host.  This should not be
   placed on the shared network volume.

5. Execute `ansible-playbook -i hosts.yml 01-deploy-ha-redis.yml
   --ask-become-pass` to deploy the HA Redis cluster.  After this step
   has completed, you may want to inspect the redis server and redis
   sentinel services now running on each machine.

