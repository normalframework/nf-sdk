Application Note: Deploying on Ubuntu
===================

Ubuntu is commonly used as a host OS.  When used for embedded applications, there are a number of "gotchas" users should be aware of in order to harden the environment to make it as reliable as possible.

**Word of warning**: the industry standard Over the Air update methodology for embedded systems requires full atomicity of system updates; for instance, using an A/B partition scheme.  Although possible with Ubuntu, setting it up is beyond the scope of this application note.  We recommend using a system like [Balena](https://www.balena.io), [Mender](https://mender.io), or other robust and tested firmware platform.  These instructions are intended to a provide a guide to the "next best" option when deploying on a server OS like Ubuntu is unavoidable.

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
By default, Ubuntu may periodically initiate filesystem checks at boot (`fsck`).  Unfortunately this may block boot on console input ("do you want to check your system?").  This should be disabled for embedded applications.


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

Filesystem Partitioning
-----------------------

Full filesystems are a common source of outages.  We recommend setting up the system with a few different partitions.  You should ensure that the out-of-band access method is hosted on the root filesystem and will start even if the filesystems are full.  This partioning scheme allows you to create an immutable OS base without requiring too much tooling or operational pain.


| Mount Point       | Type  | FS / Backend | Size (Typical)          | Options / Notes                                                          |
| ----------------- | ----- | ------------ | ----------------------- | ------------------------------------------------------------------------ |
| `/boot`           | Disk  | ext2/4       | 256–512 MB              | Mounted `ro`. Holds kernel/initramfs. Keep small and simple.             |
| `/` (rootfs)      | Disk  | ext4         | 4–8 GB                  | Mounted `ro` with **overlayroot**. Immutable OS base.                    |
| `/var`            | Disk  | ext4         | 2–4 GB (or bigger)      | Persistent. Stores system state, package DB, configs, some service data. |
| `/var/lib/docker` | Disk  | ext4 or xfs  | 4–16 GB+                | Persistent. Dedicated to Docker images/containers. Use `noatime`.        |
| `/data` *(opt)*   | Disk  | ext4         | As needed               | For app data/configs outside of Docker.                                  |
| `/tmp`            | tmpfs | RAM          | \~256–512 MB            | Ephemeral runtime scratch space.                                         |
| `/var/log`        | tmpfs | RAM          | \~128–512 MB            | Ephemeral logs. Use log forwarding if you need persistence.              |
| `/var/tmp`        | tmpfs | RAM          | \~128–512 MB            | Ephemeral temp files.                                                    |
| `/run`            | tmpfs | RAM          | Auto-managed by systemd | Holds runtime state (sockets, PID files).                                |



Configure Overlayfs:

```
$ sudo apt-get update
$ sudo apt-get install overlayroot

# in /etc/overlayroot.conf:
overlayroot="tmpfs"

# regenerate initramfs
$ sudo update-initramfs -u

```


If you need to make changes to the root filesystem, you will need to remount read-write: 
```
sudo mount -o remount,rw /
```

Systemd Watchdog
----------------
If your hardware supports it, you can configure systemd to "ping" the hardware watchdog periodically.  This will cause your board to automatically reboot if the kernel panics, etc.

It is also possible to have the watchdog monitor individual services, but a full guideline for this is beyond the scope of this note.

```
$ sudo systemctl edit system.conf

# add or uncomment these lines:
[Manager]
# Time systemd gives itself to prove it's alive before HW reset
# RuntimeWatchdogSec must be less than the hardware watchdog timeout
RuntimeWatchdogSec=30s
# Optional: reboot automatically if systemd fails to start services
RebootWatchdogSec=1min

# then
$ sudo systemctl daemon-reexec
```

Network Stickiness
------------------

Generally on embedded systems, network manager or netplan should try to activiate network connections "forever," rather than giving up.  This prevents connectivity outages due to down DHCP servers.

By default Ubuntu Server uses Netplan with the systemd-networkd backend.  Newer versions are already configured to "try forever", but you may want to mask the systemd unit that blocks boot until the device is online:

```
sudo systemctl mask systemd-networkd-wait-online.service
```
