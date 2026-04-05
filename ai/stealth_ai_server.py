#!/usr/bin/env python3
import json
import logging
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import urlparse

from stealth_ai import StealthAI  # reuse behavior from stealth_ai.py

# Global AI instance, initialized at server startup to avoid reloading the model
AI_INSTANCE = None

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        parsed = urlparse(self.path)
        if parsed.path != '/analyze':
            self.send_response(404)
            self.end_headers()
            return
        length = int(self.headers.get('content-length', 0))
        body = self.rfile.read(length)
        try:
            data = json.loads(body.decode('utf-8'))
            packet = data.get('packet', '')
            # Use the shared AI instance
            global AI_INSTANCE
            if AI_INSTANCE is None:
                # as a safe fallback, instantiate if not present
                AI_INSTANCE = StealthAI()
            score = AI_INSTANCE.analyze_traffic(packet)
            out = json.dumps({'score': score}).encode('utf-8')
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Content-Length', str(len(out)))
            self.end_headers()
            self.wfile.write(out)
        except Exception:
            logging.exception('error in analyze')
            self.send_response(500)
            self.end_headers()

def run(server_class=HTTPServer, handler_class=Handler, addr='127.0.0.1', port=5001):
    logging.basicConfig(level=logging.INFO)
    # instantiate AI once at startup
    global AI_INSTANCE
    AI_INSTANCE = StealthAI()
    logging.info('StealthAI model instantiated at startup')

    server_address = (addr, port)
    httpd = server_class(server_address, handler_class)
    logging.info('Starting StealthAI server at %s:%d', addr, port)
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        pass
    httpd.server_close()

if __name__ == '__main__':
    run()
