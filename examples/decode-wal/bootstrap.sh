
virtualenv venv
. venv/bin/activate
pip3 install -r requirements.txt   --extra-index-url https://buf.build/gen/python
python3 decode.py --file ./1756230900.sz --proto-module normalgw.hpl.v1.point_pb2 --message ObserveDataUpdatesReply   --ndjson
