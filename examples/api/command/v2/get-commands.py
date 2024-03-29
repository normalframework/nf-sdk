"""List currently running commands"""

import sys
sys.path.append("../..")

from helpers import NfClient, print_response

client = NfClient()
res = client.get("/api/v2/command")
print_response(res)
