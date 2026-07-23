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


import sys
import os

try:
    import google.generativeai as genai
    genai.configure(api_key=os.environ.get('GEMINI_API_KEY'))
    model = genai.GenerativeModel('gemini-2.0-flash')
    
    def analyze(text):
        prompt = f"Analyze the following Trivy vulnerability scan results for security risks and suggest remediation steps:\n\n{text}"
        response = model.generate_content(prompt)
        requests.post(os.environ.get("SLACK_WEBHOOK"), json={"text": f"🚨 *AI Analysis:*\n{response.text}"})
        print(response.text)
except Exception as e:
    print(f"AI analysis unavailable: {e}")
