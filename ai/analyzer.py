import google.generativeai as genai
import sys, os, requests
 
genai.configure(api_key=os.environ.get("GEMINI_API_KEY"))
model = genai.GenerativeModel("gemini-1.5-flash")
 
def analyze(logs):
    prompt = f"Analyze these DevSecOps logs for the Hardened Socks Shop app. Explain the failure and suggest a fix.\n\n{logs}"
    response = model.generate_content(prompt)
    requests.post(os.environ.get("SLACK_WEBHOOK"), json={"text": f"🚨 *AI Analysis:*\n{response.text}"})
 
if __name__ == "__main__":
    analyze(sys.stdin.read())