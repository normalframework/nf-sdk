Jupyter Notebook Examples
========================

These are examples of using the Python SDK to do a few different
things using the NF native APIs.

To install, first install jupyter and add the Python SDK to its
envronment.  

```
SDKROOT=~/src/nf-sdk
$ virtualenv venv
$. venv/bin/activate
$ pip install jupyter grpcio protobuf googleapis-common-protos
$ PYTHONPATH=$SDKROOT/python jupyter notebook
```