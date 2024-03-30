import urllib.parse
import csv
import os
import sys

import requests
from dotenv import load_dotenv

load_dotenv()

LEASH_HOST = os.getenv("LEASH_HOST")

if LEASH_HOST is None:
    print("Please provide a leash host")
    sys.exit(1)

APIKEY = os.getenv("LEASH_API_KEY")
ENDPOINT = LEASH_HOST + "/api/users"

headers = {"Authorization": f"API-Key {APIKEY}"}

with open("./data/rest_members.csv", "r") as file:
    reader = csv.reader(file)
    header = next(reader)
    for row in reader:
        user = {}
        for i, field in enumerate(header):
            user[field] = row[i]

        user_id = None
        get_req = requests.get(f"{LEASH_HOST}/api/users/get/email/{user['email']}?with_holds=true", headers=headers)
        if get_req.status_code == 200:
            print(f"User {user['email']} already exists")
            data = get_req.json()
            user_id = data["ID"]
        elif get_req.status_code == 404:
            print(f"User {user['email']} does not exist")
            req = {
                "email": user["email"],
                "name": f"{user['first_name']} {user['last_name']}",
                "role": "member",
                "type": user["type"],
                "pronouns": user["pronouns"],
            }

            if req["type"] == "undergrad" or req["type"] == "grad" or req["type"] == "program":
                req["major"] = user["major"]
                req["graduation_year"] = int(user["graduation_year"])
            elif req["type"] == "employee":
                req["job_title"] = user["job_title"]
                req["department"] = user["department"]

            response = requests.post(ENDPOINT, json=req, headers=headers)
            data = response.json()
            print(data)
            user_id = data["ID"]
        else:
            print(f"Error getting user {user['email']}")
            print(get_req.text)
            continue
        
        print(user_id)
        if user_id is not None:
            if user["docusign"] == "FALSE":
                print("No docusign")
                docusign_hold = {
                    "name": "docusign",
                    "reason": "You need to sign the docusign form before you can access the space",
                    "priority": 1,
                    "resolution_link": f'{os.getenv("DOCUSIGN_BASE")}&Member_UserName={urllib.parse.quote(req["name"])}&Member_Email={urllib.parse.quote(user["email"])}',
                }

                hold_req = requests.post(f"{LEASH_HOST}/api/users/{user_id}/holds", json=docusign_hold, headers=headers)
            
            if user["orientation"] == "FALSE":
                print("No orientation")
                orientation_hold = {
                    "name": "orientation",
                    "reason": "You need to complete the orientation before you can access the space",
                    "priority": 2,
                    "resolution_link": os.getenv("ORIENTATION_LINK")
                }

                hold_req = requests.post(f"{LEASH_HOST}/api/users/{user_id}/holds", json=orientation_hold, headers=headers)