"""Serve the lightweight atlas on loopback only; open the browser for the user."""
from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
import argparse,threading,webbrowser
root=Path(__file__).resolve().parent.parent
parser=argparse.ArgumentParser();parser.add_argument('--no-open',action='store_true');parser.add_argument('--port',type=int,default=0);args=parser.parse_args()
handler=partial(SimpleHTTPRequestHandler,directory=str(root))
try:server=ThreadingHTTPServer(('127.0.0.1',args.port),handler)
except OSError as exc:
 print(f'Yerel sunucu başlatılamadı: {exc}. Bağımsız index.html dosyasını açabilirsiniz.',flush=True)
 raise SystemExit(1)
url=f'http://127.0.0.1:{server.server_port}/atlas.html'
print('\nÜlkeler Atlası: '+url+'\nDurdurmak için Ctrl+C.\n',flush=True)
if not args.no_open:threading.Timer(.5,lambda:webbrowser.open(url)).start()
try:server.serve_forever()
except KeyboardInterrupt:pass
finally:server.server_close()
