<img src="logo_nf.png" width="50%"/>

Welcome to the NF SDK.  This repository contains a `docker-compose`
file which you can use to run a local copy of NF; and examples of
using the REST API.

For more information about see our main webpage and developer documentation.

[Normal Framework](https://www.normal.dev) | [🔗  Developer Docs](https://docs2.normal.dev)

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

Do More
=======

Normal offers several pre-built integrations with other systems under permissive licenses.  These can be quickly installed using our Application SDK.

| Integration | Description | Read Data   | Write Data | System Model |  UX |
| ----------- | ----------- | ----------- | ------------ | - | - |
| [Application Template](https://github.com/normalframework/applications-template) | Starting point for new apps.  Includes example hooks for testing point writeability and Postgres import | ✔️ | | |
| [Desigo CC](https://github.com/normalframework/app-desigocc) | Retrieve data from a Desigo CC NORIS API | ✔️ | | |
| [Archilogic](https://github.com/normalframework/app-archilogic) | Display data on a floor plan | | | | ✔️ | 
| [Guideline 36](https://github.com/normalframework/gl36-demo/tree/master) | Implement certain [Guideline 36](https://www.ashrae.org/news/ashraejournal/guideline-36-2021-what-s-new-and-why-it-s-important) sequences | | ✔️ | | ✔️ |
| [Avuity](https://github.com/normalframework/avuity-integration) | Expose data from [Avuity](https://www.avuity.com) occupancy sensors as BACnet objects | ✔️ | | ✔️ | |
| [ALC](https://github.com/normalframework/alc-plugin) | Import data from WebCTRL | | | ✔️ | |
| [OPC](https://github.com/normalframework/opc-integration) | Connect to UPC-UA Servers | ✔️ | | | | 

# AWS IoT MQTT Broker Setup

This guide provides instructions for setting up an AWS IoT MQTT broker and configuring it to connect with NF (a device management or control platform).

## Prerequisites

1. **AWS Account**: Ensure you have access to an AWS account.

## Setup AWS IoT

- Navigate to the **AWS IoT Core Console**.
- Go to **All Devices** > **Things** > **Create thing**.
- Enter a name for your IoT device (e.g., `GoIoTDevice`) and click **Next**.
- Choose **Auto-generate a new certificate** and click **Next**.
- Create a new policy by clicking the **Create Policy** link:
  - Set the policy name (e.g., `GoIoTDevice`).
  - Use the policy document example provided below, replacing `{{REGION}}` with your AWS Region and `{{ACCOUNT_ID}}` with your AWS Account ID.
- Download the following:
  - **Device certificate**
  - **Public key file**
  - **Private key file**
  - **RSA 2048 bit key: Amazon Root CA 1**

## Setup NF

- Go to the **NF Console**.
- Navigate to **Settings** > **Sparkplug**.
- Select the **AWS IoT** tab.
- Enter the **AWS IoT Core Domain** and **Thing Name** configured in step #1.
- Upload the following files:
  - **RSA 2048 bit key: Amazon Root CA 1** as the **Mqtt Cafile**
  - **Device certificate** as the **Mqtt Certfile**
  - **Private key file** as the **Mqtt Keyfile**
- Click **Save** to apply the changes.

## AWS Policy Example

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "iot:Connect",
      "Resource": "arn:aws:iot:{{REGION}}:{{ACCOUNT_ID}}:client/${iot:Connection.Thing.ThingName}"
    },
    {
      "Effect": "Allow",
      "Action": [
        "iot:Publish",
        "iot:Receive",
        "iot:PublishRetain"
      ],
      "Resource": [
        "arn:aws:iot:{{REGION}}:{{ACCOUNT_ID}}:topic/spBv1.0/normalgw/*/${iot:Connection.Thing.ThingName}",
        "arn:aws:iot:{{REGION}}:{{ACCOUNT_ID}}:topic/spBv1.0/normalgw/*/${iot:Connection.Thing.ThingName}/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": "iot:Subscribe",
      "Resource": [
        "arn:aws:iot:{{REGION}}:{{ACCOUNT_ID}}:topicfilter/spBv1.0/normalgw/*/${iot:Connection.Thing.ThingName}",
        "arn:aws:iot:{{REGION}}:{{ACCOUNT_ID}}:topicfilter/spBv1.0/normalgw/*/${iot:Connection.Thing.ThingName}/*"
      ]
    }
  ]
}
```