import json

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
            user["pronouns"] = None
        user["name"] = user_ldap["cn"]
        user["affiliation"] = user_ldap["eduPersonPrimaryAffiliation"]
        if user["affiliation"] == "Student":
            if "UMAmajor" in user_ldap:
                major = user_ldap["UMAmajor"]
                if isinstance(major, list):
                    major = ", ".join(major)
                
                user["major"] = major
                if "(AS)" in major or "(B" in major:
                    user["role"] = "undergrad"
                elif "(PhD)" in major or "(M" in major:
                    user["role"] = "grad"
                else:
                    user["role"] = "program"
            else:
                user["major"] = "UNKNOWN"
                user["role"] = "program"
        elif user["affiliation"] == "Employee":
            user["role"] = "employee"
            if "title" in user_ldap:
                user["title"] = user_ldap["title"]
            else:
                user["title"] = "UNKNOWN"
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
        
        