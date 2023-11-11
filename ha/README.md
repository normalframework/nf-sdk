
High Availability Playbooks
===========================

These instructions are tested on Ubuntu Server 22; but should be able
to be easily adapted to other environments.  They are desgined to
automate some of the more common administration tasks on a NF/HA
cluster.

1. Make sure `/data/` is a persistant local filesystem on each node.  The
   glusterfs bricks and redis data will be placed there.

2. Edit hosts.yml to provide the correct IP addresses and usernames to
   communicate with the NF nodes.  Do not rename the nodes since they
   are used in configuration templates.

3. Execute `ansible-playbook -i hosts.yml 00-deploy-system-dependencies.yml --ask-become-pass`
   if needed to install system dependencies like docker and the required
   python mdules for ansible.

5. Execute `ansible-playbook -i hosts.yml 01-deploy-glusterfs.yml
   --ask-become-pass` to configure the glusterfs shared storage, and  mount
   on all nodes.  After this step, `/var/nf` on all nodes should be a shared, replicated volume.

6. Executre `ansible-playbook -i hosts.yml 02-deploy-normal.yml --ask-become-pass`
   deploy Normal.  After this step
   has completed, you may want to inspect the redis server and redis
   sentinel services now running on each machine.  Normal will be running on `nfha-1` initially.


Further steps include integrating with DNS or a load balancer for automated failover.
