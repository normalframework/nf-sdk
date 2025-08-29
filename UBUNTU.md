Deploying on Ubuntu
===================

Ubuntu is commonly used as a host OS.  When used for embedded applications, there are a number of "gotchas" users should be aware of in order to harden the environment to make it as reliable as possible.

Resource Limits
---------------

Resource (memory or disk) is a common reason for system lockup.

Ensure that the `mem_limit` and `cpus` section of the docker-compose file for Normal impose reasonable resource limits.

Logging
-------

The docker system may not roll over its logs by default, leading to a full filesystem.  The best way to fix this is by changing the global logging configuration in `/etc/docker/daemon.json`.  After changing this file, restart the docker service using `sudo systemctl restart docker`.

```
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3" 
  }
}
```

Out-of-Band Management
----------------------

It is never a good idea to rely solely on the container tunnel for remote access; since it is impossible to view Docker logs or perform other system admistration tasks from within the container.  Good options for alternative modes of access are:
* Direct SSH access
* Tailscale
* OpenVPN

These services should be configured to run at boot, to continue retrying even when errors are encountered.

Filesystem Checks
-----------------
By default, Ubuntu may periodically initiate filesystem checks at boot (`fsck`).  Unfortunatyl this may block boot on console input ("do you want to check your system?").  This should be disabled for embedded applications.


1. Disable time-based checks for each `ext4` filesystem:
```
sudo tune2fs -i 0 -c 0 /dev/sdX
```

2. Grub / kernel safety: always run fsck repair and retry on panic.
```
# in /etc/default/grub:
GRUB_CMDLINE_LINUX_DEFAULT="quiet fsck.repair=yes panic=10"
# then run
sudo update-grub
```

3. Update /etc/fstab: set option `fsck.repair=yes` for each persistent filesystem.
