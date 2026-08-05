#import google.generativeai as genai
#import sys, os, requests
 
#genai.configure(api_key=os.environ.get("GEMINI_API_KEY"))
#model = genai.GenerativeModel("gemini-2.0-flash")
 
#def analyze(logs):
#    prompt = f"Analyze these DevSecOps logs for the Hardened Socks Shop app. Explain the failure and suggest a fix.\n\n{logs}"
#    response = model.generate_content(prompt)
#    requests.post(os.environ.get("SLACK_WEBHOOK"), json={"text": f"🚨 *AI Analysis:*\n{response.text}"})
 
#if __name__ == "__main__":
#    analyze(sys.stdin.read())


#import sys
#import os

#try:
    #import google.generativeai as genai
    #genai.configure(api_key=os.environ.get('GEMINI_API_KEY'))
    #model = genai.GenerativeModel('gemini-2.0-flash')
    
    #def analyze(text):
     #   prompt = f"Analyze the following Trivy vulnerability scan results for security risks and suggest remediation steps:\n\n{text}"
     #  response = model.generate_content(prompt)
     #   requests.post(os.environ.get("SLACK_WEBHOOK"), json={"text": f"🚨 *AI Analysis:*\n{response.text}"})
     #   print(response.text)
#except Exception as e:
 #   print(f"AI analysis unavailable: {e}")

import os
import sys
import json
import requests

FASTAPI_URL = os.getenv(
    "FASTAPI_URL",
    "http://192.168.189.128:8000/cicd-alert"
)


def analyze(trivy_report):

    sonar_report = []

    if os.path.exists("sonar-report.json"):
        try:
            with open("sonar-report.json", "r") as f:
                report = json.load(f)
                sonar_report = report.get("issues", [])
        except Exception as e:
            print(f"Unable to read Sonar report: {e}")

    payload = {
        "pipeline": os.getenv("JOB_NAME", "Sock-Shop"),
        "stage": os.getenv("STAGE_NAME", "Unknown"),
        "status": os.getenv("BUILD_STATUS", "RUNNING"),
        "sonar_report": sonar_report,
        "trivy_report": trivy_report
    }

    try:

        response = requests.post(
            FASTAPI_URL,
            json=payload,
            timeout=120
        )

        response.raise_for_status()

        try:
            result = response.json()
        except Exception:
            print("Invalid JSON received from FastAPI")
            print(response.text)
            sys.exit(1)

        print(result)

        if result.get("status") == "FAIL":
            print(result.get("reason", "Pipeline blocked"))
            sys.exit(1)

        print(result.get("reason", "Pipeline approved"))
        sys.exit(0)

    except Exception as e:

        print(f"Unable to contact FastAPI: {e}")
        sys.exit(1)


if __name__ == "__main__":

    trivy_report = sys.stdin.read()

    analyze(trivy_report)
