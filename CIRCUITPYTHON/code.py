import time
import math
import analogio
import board
import wifi
import socketpool
import adafruit_requests
import microcontroller
from adafruit_httpserver import Server, Request, Response, FileResponse, JSONResponse

# === Termistor config ===
VCC = 3.3
SERIES_RESISTOR = 100000
R0 = 100000
T0 = 25 + 273.15
BETA = 3950

adc1 = analogio.AnalogIn(board.A0)  # T1
adc2 = analogio.AnalogIn(board.A1)  # T2

def steinhart_temperature_C(adc):
    raw = adc.value
    voltage = raw * VCC / 65535
    if voltage <= 0 or voltage >= VCC:
        return None
    resistance = (VCC - voltage) * SERIES_RESISTOR / voltage
    temp_k = 1 / (1 / T0 + (1 / BETA) * math.log(resistance / R0))
    return temp_k - 273.15

# === Conectare WiFi ===
print("Connecting to WiFi...")
wifi.radio.connect("WiFi_Ray", "15936202")
print("Connected:", wifi.radio.ipv4_address)

# === Server HTTP ===
pool = socketpool.SocketPool(wifi.radio)
server = Server(pool, "/static", debug=True)

@server.route("/api/temps")
def temps(request: Request):
    t1 = steinhart_temperature_C(adc1)
    t2 = steinhart_temperature_C(adc2)
    return JSONResponse(request, {
        "sensors": [
            {"name": "T1", "temperature_celsius": round(t1, 2) if t1 is not None else None},
            {"name": "T2", "temperature_celsius": round(t2, 2) if t2 is not None else None}
        ]
    })

@server.route("/api/health")
def health(request: Request):
    return JSONResponse(request, {
        "status": "ok",
        "platform": "rp2040",
        "ip": str(wifi.radio.ipv4_address),
        "uptime": time.monotonic()
    })

@server.route("/")
def index(request: Request):
    return FileResponse(request, "/index.html")

print("Starting server on http://%s" % wifi.radio.ipv4_address)
server.serve_forever(str(wifi.radio.ipv4_address))
