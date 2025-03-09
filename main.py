import time
import os
import csv
import requests
import concurrent.futures
from playwright.sync_api import sync_playwright
from requests.auth import HTTPBasicAuth

newsletter_api_base = os.environ['NEWSLETTER_API_BASE']
edgeservpos_license_base_url = os.environ['EDGESERVPOS_LICENSE_BASE_URL']
edgeservpos_alt_base_url = os.environ['EDGESERVPOS_ALT_BASE_URL']

edgeservpos_email = os.environ['EDGESERVPOS_EMAIL']
edgeservpos_password = os.environ['EDGESERVPOS_PASSWORD']
edgeservpos_restaurant_code = os.environ['EDGESERVPOS_RESTAURANT_CODE']

newsletter_api_username = os.environ['NEWSLETTER_API_USERNAME']
newsletter_api_password = os.environ['NEWSLETTER_API_PASSWORD']

tmp_dir = os.environ['TMP_DIR']

def is_valid_email(email, invalid_chars={",", " ", "!", "#", "$", "%", "&", "*", "(", ")"}):
    """
    Check if an email contains any invalid characters.
    
    Args:
    - email (str): The email address to validate.
    - invalid_chars (set): A set of invalid characters.
    
    Returns:
    - bool: True if the email is valid, False otherwise.
    """
    return not any(char in email for char in invalid_chars)

def download_report(download_dir, file_name):
    os.makedirs(download_dir, exist_ok=True)
    
    with sync_playwright() as p:
        try:
            browser = p.chromium.launch(headless=True)
            context = browser.new_context()
            page = context.new_page()

            login_url = edgeservpos_license_base_url+"license/"
            print(f"Navigating to: {login_url}")
            page.goto(login_url)

            # Fill in login form and submit
            page.get_by_label("Email").fill(edgeservpos_email)
            page.get_by_label("Password").fill(edgeservpos_password)
            page.locator("button[aria-label='LOG IN']").click()

            # Wait for navigation to dashboard
            page.wait_for_url(edgeservpos_alt_base_url+edgeservpos_restaurant_code+"/boh/kpi-dashboard")

            time.sleep(0.5)

            # Navigate to Reports
            span = page.locator("span:has-text('Reports')")
            span.locator("xpath=..").click()

            time.sleep(0.25)
            page.click("text=Guest")
            time.sleep(0.25)

            # Click on "Guest Information Report"
            page.get_by_role("link", name="Guest Information Report").click()

            # Wait for navigation
            page.wait_for_url(edgeservpos_alt_base_url+edgeservpos_restaurant_code+"/boh/report*")

            # Click the "Filter By Agreement" dropdown
            page.locator("mat-select[formcontrolname='value']").click()
            page.get_by_role("option", name="E-Mail").wait_for()

            # Select the "E-Mail" option
            page.get_by_role("option", name="E-Mail").click()
            print("Successfully selected E-Mail filter!")

            # Initiate the file download
            with page.expect_download() as download_info:
                page.locator("button:has-text('Generate CSV')").click()

            download = download_info.value
            save_path = os.path.join(download_dir, file_name)
            download.save_as(save_path)

            print(f"File downloaded successfully: {save_path}")

        except Exception as e:
            print(f"Error: {e}")

        finally:
            browser.close()

def process_email(newsletter_user_endpoint, email, first_name, last_name, auth):
    if not is_valid_email(email):
        print(f"{email} is invalid, skipping...")
        return

    print(f"Checking if {email} is enrolled...")
    response = requests.get(newsletter_user_endpoint + email, auth=auth)

    if response.status_code == 200:
        print(f"{email} already enrolled, skipping...")
    else:
        print(f"{email} not enrolled, enrolling...")
        data = {
            "email": email,
            "first_name": first_name,
            "last_name": last_name,
            "lists": [{"id": 1, "value": 1}],
            "status": "confirmed"
        }
        response = requests.put(newsletter_user_endpoint + email, auth=auth, json=data)

        if response.status_code == 201:
            print(f"{email} successfully enrolled!")
        else:
            print(f"Request failed with status code: {response.status_code}")
            print("Response:", response.text)

def process_report(download_dir, file_name, max_workers=10):
    input_file = os.path.join(download_dir, file_name)
    newsletter_user_endpoint = newsletter_api_base + "subscribers/"
    auth = HTTPBasicAuth(newsletter_api_username, newsletter_api_password)
    
    with open(input_file, "r", encoding="utf-8", errors="ignore") as infile:
        reader = csv.reader(infile)
        
        with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
            futures = []
            for row in reader:
                if len(row) >= 3 and "@" in row[2]:
                    cleaned_row = [col.strip() for col in row[:3]]
                    first_name, last_name, email = cleaned_row
                    futures.append(executor.submit(process_email, newsletter_user_endpoint, email, first_name, last_name, auth))
            
            # Wait for all tasks to complete
            concurrent.futures.wait(futures)

if __name__ == "__main__":
	file_name = "guest-information-report.csv"
	download_report(tmp_dir, file_name)
	process_report(tmp_dir, file_name)
