import csv
import os
import sys

import requests
from dotenv import load_dotenv
import json

import urllib.parse
load_dotenv()

LEASH_HOST = os.getenv("LEASH_HOST")

if LEASH_HOST is None:
    print("Please provide a leash host")
    sys.exit(1)

APIKEY = os.getenv("LEASH_API_KEY")
ENDPOINT = LEASH_HOST + "/api/users"

headers = {"Authorization": f"API-Key {APIKEY}"}

EMAIL_INDEX = 3
FIRST_NAME_INDEX = 4
LAST_NAME_INDEX = 5
MAJOR = 6
GRADUATION_YEAR = 12
DOCUSIGN = 18
ORIENTATION = 19

users = {}

with open("data/ldap.json") as f:
    data = json.load(f)

    for netid in data:
        user_ldap = data[netid]
        email = user_ldap["mail"]
        user = {}
        user["email"] = email
        if "UMApronouns" in user_ldap:
            pronoun = user_ldap["UMApronouns"].split(" ")[0]
            if pronoun == "any":
                user["pronouns"] = "any/all"
            elif pronoun == "name":
                user["pronouns"] = "name only"
            else:
                user["pronouns"] = pronoun
        else:
            user["pronouns"] = "UNKNOWN"
        user["name"] = user_ldap["cn"]
        user["affiliation"] = user_ldap["eduPersonPrimaryAffiliation"]
        if user["affiliation"] == "Student":
            if "UMAmajor" in user_ldap:
                major = user_ldap["UMAmajor"]
                if isinstance(major, list):
                    major = ", ".join(major)
                
                user["major"] = major
                if "(AS)" in major or "(B" in major or "PR-" in major:
                    user["type"] = "undergrad"
                elif "(PhD)" in major or "(M" in major:
                    user["type"] = "grad"
                else:
                    user["type"] = "program"
            else:
                user["major"] = "UNKNOWN"
                user["type"] = "program"
        elif user["affiliation"] == "Employee":
            user["type"] = "employee"
            if "title" in user_ldap:
                user["job_title"] = user_ldap["title"]
            else:
                user["job_title"] = "UNKNOWN"
            if "departmentNumber" in user_ldap:
                department = user_ldap["departmentNumber"]
                if isinstance(department, list):
                    department = ", ".join(department)
                user["department"] = department
            else:
                user["department"] = "UNKNOWN"
        else:
            print(user["affiliation"])
        
        users[email] = user
        
with open("./data/ldap_members.csv", "r") as file:
    reader = csv.reader(file)
    header = next(reader)
    for row in reader:
        email = row[EMAIL_INDEX]
        if email not in users:
            continue
        user = users[email]
        graduation_year = row[GRADUATION_YEAR]
        docusign = row[DOCUSIGN]
        orientation = row[ORIENTATION]

        if user["type"] == "undergrad" or user["type"] == "grad" or user["type"] == "program":
            try:
                user["graduation_year"] = int(graduation_year)
            except:
                user["graduation_year"] = 2026

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
                "name": user["name"],
                "role": "member",
                "type": user["type"],
                "pronouns": user["pronouns"],
            }

            print(user["type"])
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
            if docusign == "FALSE":
                print("No docusign")
                docusign_hold = {
                    "name": "docusign",
                    "reason": "You need to sign the docusign form before you can access the space",
                    "priority": 1,
                    "resolution_link": f'{os.getenv("DOCUSIGN_BASE")}&Member_UserName={urllib.parse.quote(user["name"])}&Member_Email={urllib.parse.quote(user["email"])}',
                }

                hold_req = requests.post(f"{LEASH_HOST}/api/users/{user_id}/holds", json=docusign_hold, headers=headers)
            
            if orientation == "FALSE":
                print("No orientation")
                orientation_hold = {
                    "name": "orientation",
                    "reason": "You need to complete the orientation before you can access the space",
                    "priority": 2,
                    "resolution_link": os.getenv("ORIENTATION_LINK")
                }

                hold_req = requests.post(f"{LEASH_HOST}/api/users/{user_id}/holds", json=orientation_hold, headers=headers)