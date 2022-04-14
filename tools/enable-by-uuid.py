
import uuid
import json
import sys
import requests

def makePoint(period, u):
    return {
        "layer": "hpl:bacnet:1",
        "uuid": u,
        "period":  {
            "seconds": period,
        }
    } 

if __name__ == '__main__':
    if len(sys.argv) < 4:
        sys.stderr.write("{} <base url> <period> <uuids ... >\n".format(sys.argv[0]))
        sys.exit(1)
    try:
        period = int(sys.argv[2])
    except:
        sys.stderr.write("invalid trending period: "+ sys.argv[2] + "\n")
        sys.exit(1)
        
    for u in sys.argv[3:]:
        try:
            uuid.UUID(u)
        except:
            sys.stderr.write("invalid uuid: "+ u + "\n")
            sys.exit(1)

    print ("Setting {} points to trend at {}s".format(len(sys.argv[3:]), period))

    points = requests.post(sys.argv[1] + "/api/v1/point/points", data=json.dumps({
        "points": list(map(lambda u: makePoint(period, u), sys.argv[3:]))
    }))
    print (points)
