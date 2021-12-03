
import json
import sys
import requests

def escape(x):
    return x.replace(".", "\\.").replace(":", "\\:")

if __name__ == '__main__':
    if len(sys.argv) !=3:
        sys.stderr.write("{} <base url> <mac>\n".format(sys.argv[0]))
        sys.exit(1)
        

    points = requests.get(sys.argv[1] + "/api/v1/point/points", data=json.dumps({
        "layer": "default",
        "query": "@attr_bacnet_mac:{{{}}}".format(escape(sys.argv[2]))}))

    print ("Delete {} points? [y/N]".format(points.json()['totalCount']))
    res = input()
    if res != 'y':
        sys.exit(0)

    count = int(points.json()["totalCount"])

    PAGE_SIZE=200
    for i in range(0, count, PAGE_SIZE):
        points = requests.get(sys.argv[1] + "/api/v1/point/points", data=json.dumps({
            "layer": "default",
            "page_size": PAGE_SIZE,
            "page_offset": i,
            "query": "@attr_bacnet_mac:{{{}}}".format(escape(sys.argv[2]))}))
        uuids = [x['uuid'] for x in points.json()['points']]

        
        res = requests.delete(sys.argv[1] + '/api/v1/point/points', data=json.dumps({
            "layer": "hpl:bacnet:1",
            "uuids": uuids}))
        print ("Deleting batch {}:".format(i), res)
