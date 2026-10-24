#!/usr/bin/env bash
# 実データを使ってAPIの操作を順番に見せる発表用スクリプト。
set -euo pipefail

BASE_URL=${BASE_URL:-http://localhost:8080}
USER_A=${USER_A:-'4aHRnScUZAA9kFeb/IcyAA=='}
USER_B=${USER_B:-'fRbfcUWzOroD4uCRdlKmBQ=='}
ROOM_NAME="demo-$(date +%Y%m%d%H%M%S)-$$"
PASSWORD='demo1234'
ROOM_ID=''
JOINED=0
TEMP_DIR=$(mktemp -d)

cleanup() {
  if [[ -n "$ROOM_ID" ]]; then
    if [[ "$JOINED" == 1 ]]; then
      curl -sS -X DELETE "$BASE_URL/rooms/$ROOM_ID/membership" -H "X-User-ID: $USER_B" >/dev/null || true
    fi
    curl -sS -X DELETE "$BASE_URL/rooms/$ROOM_ID/membership" -H "X-User-ID: $USER_A" >/dev/null || true
  fi
  rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

for cmd in curl python3; do
  command -v "$cmd" >/dev/null || {
    printf '必要なコマンドがありません: %s\n' "$cmd" >&2
    exit 1
  }
done

pause() {
  read -r _
}
heading() { printf '\033[2J\033[H\033[1;36m%s\033[0m\n\n' "$1"; }
request() {
  local method=$1 path=$2 user=$3 body=${4:-} status
  local -a args=(-sS -o "$TEMP_DIR/response.json" -w '%{http_code}' -X "$method" "$BASE_URL$path" -H "X-User-ID: $user")
  if [[ -n "$body" ]]; then args+=(-H 'Content-Type: application/json' -d "$body"); fi
  printf '$ curl'
  printf ' %q' "${args[@]:3}"
  printf '\n\n'
  status=$(curl "${args[@]}")
  printf 'HTTP %s\n' "$status"
  if [[ -s "$TEMP_DIR/response.json" ]]; then
    python3 -c 'import json,sys; print(json.dumps(json.load(open(sys.argv[1])), ensure_ascii=False, indent=2))' "$TEMP_DIR/response.json" || cat "$TEMP_DIR/response.json"
  fi
  if [[ ! "$status" =~ ^2 ]]; then
    printf '\nAPIがエラーを返しました。設定とユーザーIDを確認してください。\n' >&2
    exit 1
  fi
}
receive_recommendation() {
  python3 - "$BASE_URL" "$ROOM_ID" "$USER_A" <<'PYWS'
import base64, hashlib, json, os, socket, ssl, struct, sys, urllib.parse
base, room, user = sys.argv[1:]
u = urllib.parse.urlsplit(base)
if u.scheme not in ("http", "https") or not u.hostname:
    raise SystemExit("BASE_URLにはhttpまたはhttpsのURLを指定してください")
port = u.port or (443 if u.scheme == "https" else 80)
sock = socket.create_connection((u.hostname, port), timeout=30)
if u.scheme == "https":
    sock = ssl.create_default_context().wrap_socket(sock, server_hostname=u.hostname)
key = base64.b64encode(os.urandom(16)).decode()
path = u.path.rstrip("/") + "/rooms/" + urllib.parse.quote(room, safe="") + "/ws"
host = u.netloc
request = (f"GET {path} HTTP/1.1\r\nHost: {host}\r\n"
           f"Upgrade: websocket\r\nConnection: Upgrade\r\n"
           f"Sec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n"
           f"X-User-ID: {user}\r\n\r\n")
sock.sendall(request.encode())
stream = sock.makefile("rb")
status = stream.readline().decode("iso-8859-1").strip()
headers = {}
while True:
    line = stream.readline()
    if line in (b"\r\n", b""):
        break
    name, value = line.decode("iso-8859-1").split(":", 1)
    headers[name.lower()] = value.strip()
expected = base64.b64encode(hashlib.sha1((key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").encode()).digest()).decode()
if " 101 " not in status or headers.get("sec-websocket-accept") != expected:
    raise SystemExit(f"WebSocket接続に失敗しました: {status}")
def read_exact(n):
    data = stream.read(n)
    if len(data) != n:
        raise EOFError("WebSocketが切断されました")
    return data
def send_control(opcode, payload):
    mask = os.urandom(4)
    sock.sendall(bytes([0x80 | opcode, 0x80 | len(payload)]) + mask + bytes(b ^ mask[i % 4] for i, b in enumerate(payload)))
fragments = bytearray()
while True:
    first, second = read_exact(2)
    opcode, size = first & 15, second & 127
    if size == 126: size = struct.unpack("!H", read_exact(2))[0]
    elif size == 127: size = struct.unpack("!Q", read_exact(8))[0]
    mask = read_exact(4) if second & 128 else None
    payload = read_exact(size)
    if mask: payload = bytes(b ^ mask[i % 4] for i, b in enumerate(payload))
    if opcode == 8: raise SystemExit("推薦結果の受信前に接続が終了しました")
    if opcode == 9:
        send_control(10, payload)
        continue
    if opcode not in (0, 1): continue
    fragments.extend(payload)
    if not first & 128: continue
    event = json.loads(fragments)
    fragments.clear()
    if event.get("type") in ("recommendation.completed", "recommendation.failed"):
        print(json.dumps(event, ensure_ascii=False, indent=2))
        if event["type"] == "recommendation.failed": sys.exit(1)
        break
sock.close()
PYWS
}
json_field() { python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))[sys.argv[2]])' "$TEMP_DIR/response.json" "$1"; }
encode() { python3 -c 'import sys,urllib.parse; print(urllib.parse.quote(sys.argv[1],safe=""))' "$1"; }

heading '1. ルームを作成'
body=$(python3 -c 'import json,sys; print(json.dumps({"room_name":sys.argv[1],"password":sys.argv[2]}))' "$ROOM_NAME" "$PASSWORD")
request POST /rooms "$USER_A" "$body"
ROOM_ID=$(json_field room_id)
pause

heading '2. もう一人がルームに参加'
request POST /rooms/join "$USER_B" "$body"
JOINED=1
pause

heading '3. 1人目の視聴履歴を取得'
request GET "/rooms/$ROOM_ID/members/$(encode "$USER_A")/viewing-history?limit=3" "$USER_A"
cp "$TEMP_DIR/response.json" "$TEMP_DIR/history-a.json"
pause

heading '4. 2人目の視聴履歴を取得'
request GET "/rooms/$ROOM_ID/members/$(encode "$USER_B")/viewing-history?limit=3" "$USER_B"
cp "$TEMP_DIR/response.json" "$TEMP_DIR/history-b.json"
pause

selection() {
  python3 -c 'import json,sys; items=json.load(open(sys.argv[1]))["items"]; print(json.dumps({"included_series_ids": [x["series_id"] for x in items[int(sys.argv[2]):int(sys.argv[2])+1]]}))' "$1" "$2"
}
heading '5. 1人目が番組を選択'
request PUT "/rooms/$ROOM_ID/viewing-history" "$USER_A" "$(selection "$TEMP_DIR/history-a.json" 1)"
pause

heading '6. 2人目が選択し、推薦処理を開始'
request PUT "/rooms/$ROOM_ID/viewing-history" "$USER_B" "$(selection "$TEMP_DIR/history-b.json" 0)"
pause

heading '7. WebSocketで推薦された番組を受信'
printf 'WebSocketに接続して推薦結果を受信します。\n\n'
printf '推薦結果を待っています…\n\n'
receive_recommendation | tee "$TEMP_DIR/before.json"
pause

heading '8. 二人の好みの重みを変更'
request POST "/rooms/$ROOM_ID/preference" "$USER_A" '{"weight":0.7}'
pause

heading '9. 重み変更後の推薦結果'
printf 'WebSocketに接続して推薦結果を受信します。\n\n'
receive_recommendation | tee "$TEMP_DIR/after.json"
pause

# heading '10. 推薦結果の変化を比較'
# python3 - "$TEMP_DIR/before.json" "$TEMP_DIR/after.json" <<'PYCOMPARE'
# import json, sys
# before, after = [json.load(open(path))["data"]["items"] for path in sys.argv[1:]]
# old = {item["series_id"]: rank for rank, item in enumerate(before, 1)}
# print("変更前 → 変更後の順位（新規は NEW）\n")
# for rank, item in enumerate(after, 1):
#     previous = old.get(item["series_id"])
#     print(f"{str(previous) if previous else 'NEW':>3} → {rank:>2}  {item['series_title']}")
# removed = [item["series_title"] for item in before if item["series_id"] not in {x["series_id"] for x in after}]
# if removed:
#     print("\n推薦リストから外れた番組: " + "、".join(removed))
# if [x["series_id"] for x in before] == [x["series_id"] for x in after]:
#     print("\nこの選択では、重みを変更しても推薦順位は変わりませんでした。")
# PYCOMPARE
# pause
