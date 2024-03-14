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

EMAIL_INDEX = 3
FIRST_NAME_INDEX = 4
LAST_NAME_INDEX = 5
MAJOR = 6
GRADUATION_YEAR = 7
DOCUSIGN = 12
ORIENTATION = 13


def main():
    if len(sys.argv) < 2:
        print("Please provide a file to migrate")
        return

    file_name = sys.argv[1]
    print(f"Migrating {file_name}...")
    if not os.path.exists(file_name):
        print(f"File {file_name} does not exist")
        return

    if not file_name.endswith(".csv"):
        print("Please provide a csv file")
        return

    with open(file_name, "r") as file:
        reader = csv.reader(file)
        for row in reader:
            if row[EMAIL_INDEX].startswith("Email") or row[EMAIL_INDEX] == "":
                continue

            grad = 2022
            try:
                grad = int(row[GRADUATION_YEAR])
            except:
                grad = 2022

            major = row[MAJOR]
            if major == "":
                major = "UNKNOWN"

            user_create_data = {
                "email": row[EMAIL_INDEX],
                "name": f"{row[FIRST_NAME_INDEX]} {row[LAST_NAME_INDEX]}",
                "role": "member",
                "type": "undergrad",
                "major": major,
                "graduation_year": grad,
            }

            headers = {
                "Authorization": f"API-Key {APIKEY}",
                "Content-Type": "application/json",
            }

            response = requests.post(ENDPOINT, json=user_create_data, headers=headers)
            if response.status_code == 200:
                print(f"User {row[EMAIL_INDEX]} created")
            else:
                print(f"Error creating user {row[EMAIL_INDEX]}")
                print(response.status_code)
                print(response.text)
                continue

            r = response.json()
            ID = r["ID"]

            if row[DOCUSIGN] == "TRUE":
                print(f"User {row[EMAIL_INDEX]} is docusigned")
                docusign_training = {"training_type": "docusign"}
                response = requests.post(
                    f"{ENDPOINT}/{ID}/trainings",
                    json=docusign_training,
                    headers=headers,
                )
                if response.status_code == 200:
                    print(f"User {row[EMAIL_INDEX]} docusign updated")
                else:
                    print(f"Error updating docusign for user {row[EMAIL_INDEX]}")
                    print(response.status_code)
                    print(response.text)
                    continue
            else:
                print(f"User {row[EMAIL_INDEX]} is not docusigned")
                docusign_hold = {
                    "hold_type": "docusign",
                    "reason": "You need to sign the docusign form before you can access the space",
                    "priority": 1,
                }
                response = requests.post(
                    f"{ENDPOINT}/{ID}/holds", json=docusign_hold, headers=headers
                )
                if response.status_code == 200:
                    print(f"User {row[EMAIL_INDEX]} docusign hold updated")
                else:
                    print(f"Error updating docusign hold for user {row[EMAIL_INDEX]}")
                    print(response.status_code)
                    print(response.text)
                    continue

            if row[ORIENTATION] == "TRUE":
                print(f"User {row[EMAIL_INDEX]} is oriented")
                orientation_training = {"training_type": "orientation"}
                response = requests.post(
                    f"{ENDPOINT}/{ID}/trainings",
                    json=orientation_training,
                    headers=headers,
                )
                if response.status_code == 200:
                    print(f"User {row[EMAIL_INDEX]} orientation updated")
                else:
                    print(f"Error updating orientation for user {row[EMAIL_INDEX]}")
                    print(response.status_code)
                    print(response.text)
                    continue
            else:
                print(f"User {row[EMAIL_INDEX]} is not oriented")
                orientation_hold = {
                    "hold_type": "orientation",
                    "reason": "You need to complete the orientation before you can access the space",
                    "priority": 2,
                }
                response = requests.post(
                    f"{ENDPOINT}/{ID}/holds", json=orientation_hold, headers=headers
                )
                if response.status_code == 200:
                    print(f"User {row[EMAIL_INDEX]} orientation hold updated")
                else:
                    print(
                        f"Error updating orientation hold for user {row[EMAIL_INDEX]}"
                    )
                    print(response.status_code)
                    print(response.text)
                    continue


if __name__ == "__main__":
    main()
