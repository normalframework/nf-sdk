"""Upload a file and set it as a logo
This example demonstrates uploading an image file and setting it as one of the branding variables.
Usage `python branding.py <path to image file> <env variable>`
E.g. `python branding.py ~/assets/logo.png FAV_ICON_FILE`
"""

import sys
sys.path.append("../..")
import base64
from helpers import NfClient, print_response

file = sys.argv[1]
env_var = sys.argv[2]

client = NfClient()

# # Upload the image
with open(file, 'rb') as image_file:
    encoded_string = base64.b64encode(image_file.read()).decode('utf-8')

upload_res = client.post('/api/v1/upload', 
    params={'name': image_file.name},
    json=encoded_string
)

print_response(upload_res)
file_token = upload_res.json()["fileToken"]

# Set the uploaded file as a configuration variable
env_res = client.post('/api/v1/platform/env', json={
    "variables": [{
        "id": env_var, #"FAV_ICON_FILE",
        "is_advanced": False,
        "is_default": False,
        "is_empty": False,
        "file": {
            "file_name": file_token,
        }
    }]
})