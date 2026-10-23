import os
import random
from datetime import datetime, timezone

from locust import HttpUser, task, between

CARNET = os.getenv("CARNET", "201800632")
ROUTE = os.getenv("ROUTE", "grpc")          # grpc | dapr
COUNTRIES = ["USA", "RUS", "CHN", "ESP", "GTM"]


class MilitaryReporter(HttpUser):
    host = os.getenv("TARGET_HOST", "http://localhost:8080")
    wait_time = between(0.01, 0.1)

    @task
    def send_report(self):
        payload = {
            "country": random.choice(COUNTRIES),
            "warplanes_in_air": random.randint(0, 50),
            "warships_in_water": random.randint(0, 30),
            "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        }
        self.client.post(f"/{ROUTE}-{CARNET}", json=payload, name=f"/{ROUTE}-[carnet]")
