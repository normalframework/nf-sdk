Python API Examples
===================

In addition to the [REST API Reference](https://docs.normal.dev/api/html/api.html), we provide certain simple examples of how to call the API in Python.  This includes support for an OAuth client to make it easy to use against the public tunnel enpoints provided as part of the [Normal Portal](https://portal.normal-online.net).

Setup
-----

The examples all use a wrapper around the Python requests module in [helpers.py](./helpers.py).  There are three important environment variables:

* `NFURL`: the base URL of the Normal installation; for instance, the tunnel URL or your local environment.
* `NF_CLIENT_ID` and `NF_CLIENT_SECRET`: the client ID and secret generated on the `Settings > API Keys` page of your Normal installation.

You may also need to install the requests module: `pip3 install requests`.

Usage
-----

Each example is relatively self contained, and simply calls the API with some example arguments and prints the result.  You may need to expect to edit some of the examples; for instance to change point IDs to refer to points available in your installation.

