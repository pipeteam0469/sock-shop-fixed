"""
FastAPI AI Alert Service
========================
Receives Prometheus Alertmanager webhook alerts, collects Kubernetes diagnostics
(pod logs, describe output, events, restart counts, Prometheus metrics),
sends everything to Google Gemini AI for analysis, and posts the structured
incident report to Slack.

Workflow:
  Alertmanager → Webhook → FastAPI → K8s Diagnostics → Gemini AI → Slack
"""

import os
import sys
import json
import logging
import subprocess
import traceback
from datetime import datetime, timezone
from typing import Optional

import requests
from fastapi import FastAPI, Request, BackgroundTasks, HTTPException
from pydantic import BaseModel, Field
import google.generativeai as genai

# ── Logging ──────────────────────────────────────────────────────────────────
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
)
logger = logging.getLogger("ai-alert-service")

# ── Environment Variables ────────────────────────────────────────────────────
GEMINI_API_KEY: str = os.environ.get("GEMINI_API_KEY", "")
GEMINI_MODEL: str = os.environ.get("GEMINI_MODEL", "gemini-2.0-flash")
SLACK_WEBHOOK: str = os.environ.get("SLACK_WEBHOOK", "")
PROMETHEUS_URL: str = os.environ.get("PROMETHEUS_URL", "http://prometheus.monitoring.svc.cluster.local:9090")
NAMESPACE: str = os.environ.get("K8S_NAMESPACE", "default")

app = FastAPI(title="PipeGuard AI Alert Service")

# ── Kubernetes helpers ───────────────────────────────────────────────────────

def _kubectl(args: list[str], timeout: int = 15) -> str:
    """Run a kubectl command and return stdout (or error string)."""
    try:
        result = subprocess.run(
            ["kubectl"] + args,
            capture_output=True, text=True, timeout=timeout,
        )
        return result.stdout.strip() or result.stderr.strip()
    except subprocess.TimeoutExpired:
        return f"[timeout after {timeout}s]"
    except Exception as exc:
        return f"[error running kubectl: {exc}]"


def collect_pod_logs(pod_name: str, namespace: str) -> str:
    """Get the last 200 lines of logs for the given pod."""
    return _kubectl([
        "logs", pod_name, "-n", namespace,
        "--tail=200", "--all-containers",
    ], timeout=20)


def collect_pod_describe(pod_name: str, namespace: str) -> str:
    """Get kubectl describe output for the given pod."""
    return _kubectl(["describe", "pod", pod_name, "-n", namespace], timeout=15)


def collect_namespace_events(namespace: str) -> str:
    """Get recent Kubernetes events for the namespace."""
    return _kubectl(["get", "events", "-n", namespace, "--sort-by=.lastTimestamp"], timeout=15)


def collect_restart_counts(namespace: str) -> str:
    """Get restart counts for all pods in the namespace."""
    return _kubectl([
        "get", "pods", "-n", namespace,
        "-o", "custom-columns=NAME:.metadata.name,RESTARTS:.status.containerStatuses[*].restartCount",
    ], timeout=10)


def collect_pod_status(namespace: str) -> str:
    """Get status and phase of all pods in the namespace."""
    return _kubectl([
        "get", "pods", "-n", namespace,
        "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase,IP:.status.podIP,NODE:.spec.nodeName",
    ], timeout=10)


def collect_prometheus_metrics(alert_name: str) -> str:
    """Query Prometheus for the current metric value of the firing alert."""
    try:
        resp = requests.get(
            f"{PROMETHEUS_URL}/api/v1/query",
            params={"query": alert_name},
            timeout=5,
        )
        if resp.status_code == 200:
            data = resp.json()
            return json.dumps(data.get("data", {}), indent=2)
        return f"[Prometheus returned {resp.status_code}]"
    except Exception as exc:
        return f"[Prometheus query error: {exc}]"


# ── Alertmanager Webhook Payload ─────────────────────────────────────────────

class AlertPayload(BaseModel):
    """Subset of the Alertmanager webhook payload we care about."""
    version: str = "4"
    groupKey: Optional[str] = None
    status: str = "firing"
    receiver: str = "ai-service"
    alerts: list[dict] = Field(default_factory=list)


# ── Gemini AI Analysis ───────────────────────────────────────────────────────

def build_ai_prompt(alert_info: str, diagnostics: str) -> str:
    """Build the structured prompt for Gemini AI."""
    return f"""You are a DevSecOps incident analysis assistant for the PipeGuard platform running the Hardened Sock Shop microservices on Kubernetes.

## INCIDENT ALERT
{alert_info}

## KUBERNETES DIAGNOSTICS
{diagnostics}

## YOUR TASK
Analyze this incident and produce a structured incident report. You MUST follow this exact format:

### Root Cause Analysis
[Explain the most likely root cause based on the evidence provided.]

### Evidence Summary
[Summarize the key evidence from logs, events, metrics, and pod status that supports your root cause analysis.]

### Impact Assessment
[Describe the current and potential impact on the Sock Shop application and its users.]

### Immediate Actions
[List the steps that should be taken RIGHT NOW to mitigate or resolve the issue.]

### Long-term Recommendations
[Suggest architectural or configuration changes to prevent this type of incident in the future.]

### Confidence Level
[State your confidence level (High/Medium/Low) and explain why.]
"""


def call_gemini(prompt: str) -> str:
    """Send the prompt to Gemini and return the response text."""
    genai.configure(api_key=GEMINI_API_KEY)
    model = genai.GenerativeModel(GEMINI_MODEL)
    response = model.generate_content(prompt)
    return response.text


# ── Slack Notification ───────────────────────────────────────────────────────

def send_slack_alert(message: str) -> None:
    """Send a formatted message to Slack via webhook."""
    if not SLACK_WEBHOOK:
        logger.warning("SLACK_WEBHOOK not configured — skipping Slack notification")
        return
    try:
        payload = {"text": message}
        resp = requests.post(SLACK_WEBHOOK, json=payload, timeout=10)
        logger.info("Slack notification sent — status %d", resp.status_code)
    except Exception as exc:
        logger.error("Failed to send Slack notification: %s", exc)


def format_slack_message(report: str, alert_name: str, severity: str) -> str:
    """Format the AI report into a Slack-friendly message."""
    severity_emoji = {"critical": "🔴", "warning": "🟡", "info": "🔵"}.get(severity, "⚪")
    timestamp = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M:%S UTC")

    # Truncate report if too long for Slack (max ~3000 chars)
    max_len = 2900
    if len(report) > max_len:
        report = report[:max_len] + "\n\n... (report truncated, check full report in logs)"

    return (
        f"{severity_emoji} *PIPEGUARD AI INCIDENT REPORT*\n"
        f"*Timestamp:* {timestamp}\n"
        f"*Alert:* {alert_name}\n"
        f"*Severity:* {severity.upper()}\n"
        f"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
        f"{report}"
    )


# ── Alert Processing ─────────────────────────────────────────────────────────

def process_alert(payload: AlertPayload) -> None:
    """
    Main alert processing pipeline:
      1. Extract alert info
      2. Collect Kubernetes diagnostics
      3. Query Prometheus metrics
      4. Send to Gemini AI
      5. Post to Slack
    """
    alerts = payload.alerts
    if not alerts:
        logger.info("No alerts in payload — skipping")
        return

    # We process the first (most critical) alert in the group
    alert = alerts[0]
    alert_name = alert.get("labels", {}).get("alertname", "Unknown")
    severity = alert.get("labels", {}).get("severity", "warning")
    pod_name = alert.get("labels", {}).get("pod", "")
    namespace = alert.get("labels", {}).get("namespace", NAMESPACE)

    logger.info("Processing alert: %s (severity=%s, pod=%s, ns=%s)", alert_name, severity, pod_name, namespace)

    # 1. Build alert info string
    alert_info = json.dumps(alert, indent=2)

    # 2. Collect Kubernetes diagnostics
    diagnostics_parts = []

    # Pod logs (only if we know the pod name)
    if pod_name:
        diagnostics_parts.append(f"### Pod Logs ({pod_name})")
        diagnostics_parts.append(collect_pod_logs(pod_name, namespace))

        # Describe pod
        diagnostics_parts.append(f"### Pod Describe ({pod_name})")
        diagnostics_parts.append(collect_pod_describe(pod_name, namespace))

    # Namespace events (always)
    diagnostics_parts.append(f"### Recent Events (namespace: {namespace})")
    diagnostics_parts.append(collect_namespace_events(namespace))

    # Pod status
    diagnostics_parts.append(f"### Pod Status (namespace: {namespace})")
    diagnostics_parts.append(collect_pod_status(namespace))

    # Restart counts
    diagnostics_parts.append(f"### Container Restart Counts (namespace: {namespace})")
    diagnostics_parts.append(collect_restart_counts(namespace))

    # 3. Prometheus metrics
    diagnostics_parts.append(f"### Prometheus Metrics Query (alert: {alert_name})")
    diagnostics_parts.append(collect_prometheus_metrics(alert_name))

    diagnostics = "\n\n".join(diagnostics_parts)

    # 4. Send to Gemini AI
    logger.info("Sending diagnostics to Gemini AI (model=%s)...", GEMINI_MODEL)
    try:
        prompt = build_ai_prompt(alert_info, diagnostics)
        ai_report = call_gemini(prompt)
        logger.info("Gemini AI analysis complete")
    except Exception as exc:
        logger.error("Gemini AI analysis failed: %s", exc)
        traceback.print_exc()
        ai_report = f"**AI Analysis Unavailable**\n\nError: {exc}\n\nPlease investigate manually."

    # 5. Post to Slack
    slack_message = format_slack_message(ai_report, alert_name, severity)
    send_slack_alert(slack_message)
    logger.info("Alert processing complete for %s", alert_name)


# ── FastAPI Endpoints ────────────────────────────────────────────────────────

@app.get("/")
async def health():
    """Health check endpoint."""
    return {"status": "healthy", "service": "PipeGuard AI Alert Service"}


@app.get("/health")
async def health_check():
    """Alternative health check endpoint for Kubernetes probes."""
    return {"status": "ok"}


@app.post("/alert")
async def receive_alert(request: Request, background_tasks: BackgroundTasks):
    """
    Receive Alertmanager webhook alerts.
    Alertmanager sends a POST with the alert group payload.
    """
    try:
        body = await request.json()
    except Exception as exc:
        raise HTTPException(status_code=400, detail=f"Invalid JSON: {exc}")

    logger.info("Received Alertmanager webhook — %d alert(s)", len(body.get("alerts", [])))

    # Process in background so Alertmanager gets a fast 200 response
    background_tasks.add_task(process_alert, AlertPayload(**body))

    return {"status": "accepted"}


@app.post("/alert/test")
async def test_alert(background_tasks: BackgroundTasks):
    """
    Test endpoint — simulates a critical alert for testing the full pipeline.
    """
    test_payload = AlertPayload(
        version="4",
        groupKey="test-group",
        status="firing",
        receiver="ai-service",
        alerts=[{
            "status": "firing",
            "labels": {
                "alertname": "HighCPUUsage",
                "severity": "critical",
                "category": "resource",
                "pod": "front-end-test",
                "namespace": "default",
            },
            "annotations": {
                "summary": "Test: High CPU usage on pod front-end-test",
                "description": "This is a test alert for verifying the AI alert pipeline.",
                "impact": "Test impact",
                "suggested_action": "Verify Slack notification is received.",
            },
            "startsAt": datetime.now(timezone.utc).isoformat(),
            "endsAt": "",
            "generatorURL": "http://prometheus:9090",
            "fingerprint": "test-fingerprint-001",
        }],
    )

    background_tasks.add_task(process_alert, test_payload)
    return {"status": "test-alert-queued"}


@app.get("/diagnostics/{pod_name}")
async def get_diagnostics(pod_name: str, namespace: str = NAMESPACE):
    """
    Manual diagnostics endpoint — collects all K8s diagnostics for a given pod.
    Useful for debugging without triggering an alert.
    """
    return {
        "pod": pod_name,
        "namespace": namespace,
        "logs": collect_pod_logs(pod_name, namespace),
        "describe": collect_pod_describe(pod_name, namespace),
        "events": collect_namespace_events(namespace),
        "restart_counts": collect_restart_counts(namespace),
        "pod_status": collect_pod_status(namespace),
    }
